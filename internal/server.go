package internal

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type poolConfig struct {
	// Maximum number of simultaneous connections
	MaxConns uint8 `json:"maxConns"`
	// Connection timout threshold in milliseconds
	TimeoutThreshold uint16 `json:"timeoutThreshold"`
	// Backup for server downtime
	BackupOn bool `json:"backUpOn"`
	// Backup cycle in milliseconds
	BackupCycle uint32 `json:"backUpCycle"`
	// Lifetime of a single cache node in milliseconds
	TimeToLive uint16 `json:"timeToLive"`
	// Maximum number of simultaneous cache nodes
	NodeLimit uint32 `json:"nodeLimit"`
	// Single cache node size maximum limit in bytes
	NodeSize uint16 `json:"nodeSize"`
	// cache size maximum limit in bytes
	CacheLimit uint32 `json:"cacheLimit"`
	// Maximum idle time in seconds before memory cleanup
	IdleThreshold uint16 `json:"idleThreshold"`
}

type ServerConfig struct {
	Name             string      `json:"name"`
	User             string      `json:"user"`
	Password         string      `json:"password"`
	Port             uint32      `json:"port"`
	EnableEncryption bool        `json:"enableEncryption"`
	PoolConfig       *poolConfig `json:"poolConfig"`
}

type credentials struct {
	User     string
	Password string
}

type socket struct {
	serverPublicKey  *rsa.PublicKey
	serverPrivateKey *rsa.PrivateKey
	logger           *Cli
	conn             net.Conn
}

type LebreServer struct {
	ServerConfig ServerConfig
	credentials  *credentials
	cache        cache
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:             5051,
		EnableEncryption: true,
		PoolConfig: &poolConfig{
			MaxConns:         15,
			TimeoutThreshold: 5000,
			BackupCycle:      300000,
			TimeToLive:       300,
			NodeLimit:        3500,
			NodeSize:         1024,
			CacheLimit:       5242880,
			IdleThreshold:    3600,
		},
	}
}

func (socket *socket) getRequestParts(data string) ([]string, error) {
	decryptedData, err := RSADecrypt([]byte(data), socket.serverPrivateKey)
	if err != nil {
		return nil, err
	}

	return strings.Fields(string(decryptedData)), nil

}

func (socket *socket) sendMessage(data []byte) error {
	var buffer bytes.Buffer

	err := binary.Write(&buffer, binary.BigEndian, uint32(len(data)))
	if err != nil {
		socket.logger.ErrLog.Printf("ERR couldn't append data length to chunk: %s\n", err)
		return err
	}

	// write public key to buffer
	_, err = buffer.Write(data)
	if err != nil {
		socket.logger.ErrLog.Println("ERR couldn't write data to buffer")
		return err
	}

	// send buffer bytes to the connection
	_, err = socket.conn.Write(buffer.Bytes())
	if err != nil {
		socket.logger.ErrLog.Println("ERR couldn't flush data from buffer to client")
		return err
	}

	buffer.Reset()
	return nil
}

func (socket *socket) respond(data string) {
	encryptedResponse, err := RSAEncrypt([]byte(data), socket.serverPublicKey)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error encrypting response: %s", err))
		return
	}

	err = socket.sendMessage(encryptedResponse)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending response: %s", err))
		return
	}
}

func (socket *socket) negotiate() error {
	// receive server public key
	readerBuffer := make([]byte, 1024)
	_, err := socket.conn.Read(readerBuffer)
	if err != nil {
		socket.logger.ErrLog.Printf("ERR couldn't read server public key from connection: %s\n", err)
		return err
	}

	var serverPublicKey *rsa.PublicKey
	serverPublicKey, err = PEMToPublicKey(strings.TrimSpace(string(readerBuffer)))
	if err != nil {
		socket.logger.ErrLog.Printf("ERR couldn't parse server pem public key: %s\n", err)
		return err
	}
	socket.serverPublicKey = serverPublicKey
	err = socket.sendMessage([]byte("SERVER RECEIVED KEY"))
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending response: %s", err))
		return err
	}

	publicKey, err := PublicKeyToPEM(&socket.serverPrivateKey.PublicKey)
	if err != nil {
		socket.logger.ErrLog.Printf("ERR attempt to encode client public key failed: %s\n", err)
		return err
	}

	err = socket.sendMessage(publicKey)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending client public key: %s", err))
		return err
	}
	return nil
}

func (lebreServer *LebreServer) newCache() {
	lebreServer.cache = cache{
		Data:            make(map[string]cacheNode),
		Capacity:        lebreServer.cache.Capacity,
		CumulativeBytes: 0,
		NodeTimeToLive:  lebreServer.cache.NodeTimeToLive,
		NodeSize:        lebreServer.cache.NodeSize,
		LimitInBytes:    lebreServer.cache.LimitInBytes,
	}
}

func (lebreServer *LebreServer) backup() {
	cacheData, err := json.MarshalIndent(lebreServer.cache, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON: ", err)
		return
	}

	err = os.WriteFile("backup.json", cacheData, 0644)
	if err != nil {
		fmt.Println("Error writing JSON to file: ", err)
		return
	}
}

func (lebreServer *LebreServer) readFromBackup() error {
	if lebreServer.ServerConfig.PoolConfig.BackupOn {
		fileData, err := os.ReadFile("backup.json")
		if err != nil {
			fmt.Println("Error reading JSON file: ", err)
			return err
		}
		err = json.Unmarshal(fileData, lebreServer.cache)
		if err != nil {
			fmt.Println("Error unmarshalling JSON: ", err)
			return err
		}

		return nil
	}

	return fmt.Errorf("backup is off")
}

func (lebreServer *LebreServer) handleConnection(
	conn net.Conn,
	semaphore chan struct{},
) {
	defer func() { <-semaphore }()
	defer conn.Close()

	authorized := false
	semaphore <- struct{}{}
	logger := NewCli()
	serverPrivateKey, _, err := GenerateRSAKeyPair()
	if err != nil {
		fmt.Println("Error generating RSA key pair:", err)
		conn.Close()
		return
	}

	socket := &socket{
		serverPrivateKey: serverPrivateKey,
		logger:           NewCli(),
		conn:             conn,
	}

	err = socket.negotiate()
	if err != nil {
		conn.Close()
		return
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		cmd := scanner.Text()
		parts, err := socket.getRequestParts(cmd)

		if err != nil {
			logger.ErrLog.Printf("Error while getting request parts: %s\n", err)
			conn.Close()
			return
		}

		if len(parts) == 0 {
			continue
		}

		if parts[0] == "AUTH" {
			password := sha256.Sum256([]byte(parts[1]))
			logger.Log(fmt.Sprintf("[REQUEST]: AUTH %s %x", parts[1], password))
		} else {
			logger.Log(fmt.Sprintf("[REQUEST]: %s", strings.Join(parts, " ")))
		}

		switch parts[0] {
		case "AUTH":
			if len(parts) != 3 {
				socket.respond("ERR wrong number of arguments for AUTH")
				continue
			}

			incomingUserHash := sha256.Sum256([]byte(parts[1]))
			incomingPasswordHash := sha256.Sum256([]byte(parts[2]))
			incomingUserHashString := hex.EncodeToString(incomingUserHash[:])
			incomingPasswordHashString := hex.EncodeToString(incomingPasswordHash[:])

			if lebreServer.credentials.User != incomingUserHashString ||
				lebreServer.credentials.Password != incomingPasswordHashString {
				socket.respond("ERR authentication faild")
				continue
			}

			logger.Log(fmt.Sprintf("[LOG]: Authenticated with user: %s\n", parts[1]))

			authorized = true

		case "SET":
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(parts) != 3 {
				socket.respond("ERR wrong number of arguments for SET")
				continue
			}
			err := lebreServer.cache.Set(parts[1], parts[2])
			if err != nil {
				socket.respond(fmt.Sprintf("ERR %s", err))
				continue
			}
			_, err = conn.Write([]byte("OK\n"))
			if err != nil {
				logger.ErrLog.Println("ERR couldn't write back to socket")
			}

		case "GET":
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(parts) != 2 {
				socket.respond("ERR wrong number of arguments for GET")
				continue
			}
			value, ok := lebreServer.cache.Get(parts[1])
			if ok {
				socket.respond(fmt.Sprintf("VALUE %s", value))
			} else {
				socket.respond("NOT_FOUND")
			}

		case "DELETE":
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(parts) != 2 {
				socket.respond("ERR wrong number of arguments for DELETE")
				continue
			}
			lebreServer.cache.Delete(parts[1])
			socket.respond("OK")

		default:
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			socket.respond("ERR unknown verb")
		}
	}
}

func (lebreServer *LebreServer) Start() {
	err := lebreServer.readFromBackup()
	if err != nil {
		lebreServer.newCache()
	}

	cli := NewCli()
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", lebreServer.ServerConfig.Port))
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %s", err))
		return
	}
	defer listener.Close()

	lebreServer.credentials = &credentials{
		User:     lebreServer.ServerConfig.User,
		Password: lebreServer.ServerConfig.Password,
	}
	semaphore := make(chan struct{}, lebreServer.ServerConfig.PoolConfig.MaxConns)

	if lebreServer.ServerConfig.PoolConfig.BackupOn {
		interval := time.Duration(lebreServer.ServerConfig.PoolConfig.BackupCycle) * time.Millisecond
		go Interval(interval, lebreServer.backup)
	}

	cli.Launch(fmt.Sprintf("Lebre cache server initiated. Listening on port %d", lebreServer.ServerConfig.Port))

	for {
		conn, err := listener.Accept()
		if err != nil {
			cli.Error(fmt.Sprintf("Error accepting connection: %s", err))
			continue
		}
		go lebreServer.handleConnection(conn, semaphore)
	}
}
