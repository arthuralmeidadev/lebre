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
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type poolConfig struct {
	// Maximum number of simultaneous connections
	MaxConns uint8 `json:"maxConns"`
	// Connection timout threshold in milliseconds
	ConnectionTimeout uint16 `json:"connectionTimeout"`
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
			MaxConns:          15,
			ConnectionTimeout: 30000,
			BackupCycle:       300000,
			TimeToLive:        300,
			NodeLimit:         3500,
			NodeSize:          1024,
			CacheLimit:        5242880,
		},
	}
}

func (socket *socket) getRequestParts(data []byte) ([]string, error) {
	decryptedData, err := RSADecrypt(data, socket.serverPrivateKey)
	if err != nil {
		return nil, err
	}

	return strings.Fields(string(decryptedData)), nil

}

func (socket *socket) sendMessage(data []byte) error {
	var buffer bytes.Buffer

	err := binary.Write(&buffer, binary.BigEndian, uint32(len(data)))
	if err != nil {
		return fmt.Errorf("ERR couldn't append data length to chunk: %s", err)
	}

	// write public key to buffer
	_, err = buffer.Write(data)
	if err != nil {
		return fmt.Errorf("ERR couldn't write data to buffer: %s", err)
	}

	// send buffer bytes to the connection
	_, err = socket.conn.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("ERR couldn't flush data from buffer to client: %s", err)
	}

	buffer.Reset()
	return nil
}

func (socket *socket) respond(data string) {
	encryptedResponse, err := RSAEncrypt([]byte(data), socket.serverPublicKey)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error encrypting response: %s\n", err))
		return
	}

	err = socket.sendMessage(encryptedResponse)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending response: %s\n", err))
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
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending response: %s\n", err))
		return err
	}

	publicKey, err := PublicKeyToPEM(&socket.serverPrivateKey.PublicKey)
	if err != nil {
		socket.logger.ErrLog.Printf("ERR attempt to encode client public key failed: %s\n", err)
		return err
	}

	err = socket.sendMessage(publicKey)
	if err != nil {
		socket.logger.ErrLog.Println(fmt.Sprintf("Error while sending client public key: %s\n", err))
		return err
	}
	return nil
}

func (lebreServer *LebreServer) newCache() {
	lebreServer.cache = cache{
		Data:            make(map[string]cacheNode),
		Capacity:        lebreServer.ServerConfig.PoolConfig.NodeLimit,
		CumulativeBytes: 0,
		NodeTimeToLive:  lebreServer.ServerConfig.PoolConfig.TimeToLive,
		NodeSize:        lebreServer.ServerConfig.PoolConfig.NodeSize,
		LimitInBytes:    lebreServer.ServerConfig.PoolConfig.NodeLimit,
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

	ConnectionTimeout := time.Duration(lebreServer.ServerConfig.PoolConfig.ConnectionTimeout)
	conn.SetDeadline(time.Now().Add(time.Millisecond * ConnectionTimeout))

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

	// var buffer bytes.Buffer
	reader := bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * ConnectionTimeout))
		lengthBytes := make([]byte, 4)
		_, err := io.ReadFull(reader, lengthBytes)
		if err != nil {
			logger.ErrLog.Printf("ERR failed to read message length: %s\n", err)
			return
		}

		// retrieve first 32 bits containing the message length integer
		messageLength := binary.BigEndian.Uint32(lengthBytes)
		messageBytes := make([]byte, messageLength)
		_, err = io.ReadFull(reader, messageBytes)
		if err != nil {
			logger.ErrLog.Printf("ERR failed to read message: %s\n", err)
			return
		}
		requestParts, err := socket.getRequestParts(messageBytes)
		if err != nil {
			logger.ErrLog.Printf("ERR couldn't get request parts: %s\n", err)
			conn.Close()
			return
		}

		commandParts := requestParts[1:]

		if len(commandParts) == 0 {
			continue
		}

		switch commandParts[0] {
		case "AUTH":
			logger.Log(fmt.Sprintf("[REQUEST]: %s %x", strings.Join(requestParts[0:2], " "), sha256.Sum256([]byte(requestParts[3]))))
			if len(commandParts) != 3 {
				socket.respond("ERR wrong number of arguments for AUTH")
				continue
			}

			incomingUserHash := sha256.Sum256([]byte(commandParts[1]))
			incomingPasswordHash := sha256.Sum256([]byte(commandParts[2]))
			incomingUserHashString := hex.EncodeToString(incomingUserHash[:])
			incomingPasswordHashString := hex.EncodeToString(incomingPasswordHash[:])

			if lebreServer.credentials.User != incomingUserHashString ||
				lebreServer.credentials.Password != incomingPasswordHashString {
				socket.respond("ERR authentication faild")
				continue
			}

			logger.Log(fmt.Sprintf("[LOG]: Authenticated with user: %s", commandParts[1]))

			authorized = true

		case "SET":
			logger.Log(fmt.Sprintf("[REQUEST]: %s %x", strings.Join(requestParts[0:3], " "), sha256.Sum256([]byte(requestParts[3]))))
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(commandParts) != 3 {
				socket.respond("ERR wrong number of arguments for SET")
				continue
			}
			value := strings.ReplaceAll(commandParts[2], "\\u0020", "\u0020")
			err := lebreServer.cache.Set(commandParts[1], value)
			if err != nil {
				socket.respond(fmt.Sprintf("ERR %s", err))
				continue
			}

			socket.respond("OK")

		case "GET":
			logger.Log(fmt.Sprintf("[REQUEST]: %s", strings.Join(requestParts, " ")))
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(commandParts) != 2 {
				socket.respond("ERR wrong number of arguments for GET")
				continue
			}
			value, ok := lebreServer.cache.Get(commandParts[1])
			if ok && len(value) > 0 {
				socket.respond(fmt.Sprintf("VALUE %s", value))
			} else {
				socket.respond("NOT_FOUND")
			}

		case "DELETE":
			logger.Log(fmt.Sprintf("[REQUEST]: %s", strings.Join(requestParts, " ")))
			if !authorized {
				socket.respond("ERR unauthorized")
				continue
			}
			if len(commandParts) != 2 {
				socket.respond("ERR wrong number of arguments for DELETE")
				continue
			}
			lebreServer.cache.Delete(commandParts[1])
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
