package internal

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
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
	// Cache size maximum limit in bytes
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

type cache struct {
	Data            map[string]cacheNode `json:"data"`
	Capacity        uint32               `json:"capacity"`
	CumulativeBytes uint32               `json:"cumulativeBytes"`
	NodeTimeToLive  uint16               `json:"nodeTimeToLive"`
	NodeSize        uint16               `json:"nodeSize"`
	LimitInBytes    uint32               `json:"limitInBytes"`
	Mutex           sync.RWMutex         `json:"-"`
}

type cacheNode struct {
	Value  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

type credentials struct {
	User     string
	Password string
}

type client struct {
	serverPublicKey *rsa.PublicKey
	clientPublicKey *rsa.PublicKey
	privateKey      *rsa.PrivateKey
	logger          *Cli
	conn            net.Conn
}

func (client *client) respond(data string) {
	encryptedResponse, err := RSAEncrypt([]byte(data+"\n"), client.serverPublicKey)
	if err != nil {
		fmt.Println("Error encrypting: ", err)
		return
	}

	_, err = client.conn.Write(encryptedResponse)
	if err != nil {
		client.logger.ErrLog.Println("ERR couldn't write back to client")
	}
}

func (client *client) negotiate() {
	_, err := client.conn.Write([]byte(fmt.Sprintf("PUBKEY %s", client.clientPublicKey.N.String())))
	if err != nil {
		client.logger.ErrLog.Println("ERR couldn't send public key to client")
	}

	buffer := make([]byte, 1024)
	n, err := client.conn.Read(buffer)
	if err != nil {
		return
	}

	var clientPublicKey rsa.PublicKey
	err = json.Unmarshal(buffer[:n], &clientPublicKey)
	if err != nil {
		return
	}

	client.clientPublicKey = &clientPublicKey
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

func newCache(
	capacity uint32,
	limitInBytes uint32,
	nodeTimeToLive uint16,
	nodeSize uint16,
) *cache {
	return &cache{
		Data:            make(map[string]cacheNode),
		Capacity:        capacity,
		CumulativeBytes: 0,
		NodeTimeToLive:  nodeTimeToLive,
		NodeSize:        nodeSize,
		LimitInBytes:    limitInBytes,
	}
}

func (cache *cache) Set(key, value string) error {
	now := time.Now()
	cacheNode := cacheNode{
		Value:  value,
		Expiry: now.Add(300 * time.Second),
	}

	incomingDataByteSize := len(key) + len(value)

	if incomingDataByteSize > int(cache.NodeSize) {
		return fmt.Errorf("node byte limit exceeded. Max is: %d", cache.NodeSize)
	}

	if incomingDataByteSize+int(cache.CumulativeBytes) > int(cache.LimitInBytes) {
		return fmt.Errorf("cache byte limit exceeded. Max is: %d", cache.LimitInBytes)
	} else {
		cache.CumulativeBytes = uint32(incomingDataByteSize) + cache.CumulativeBytes
	}

	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	delete(cache.Data, key)
	cache.Data[key] = cacheNode

	if len(cache.Data) > int(cache.Capacity) {
		for keyToBeDeleted := range cache.Data {
			delete(cache.Data, keyToBeDeleted)
			break
		}
	}
	return nil
}

func (cache *cache) Get(key string) (string, bool) {
	cache.Mutex.RLock()
	defer cache.Mutex.RUnlock()

	node, ok := cache.Data[key]

	if node.Expiry.After(time.Now()) {
		delete(cache.Data, key)
		return "", true
	}

	return node.Value, ok
}

func (cache *cache) Delete(key string) {
	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	delete(cache.Data, key)
}

func handleClient(
	conn net.Conn,
	cache *cache,
	credentials *credentials,
	semaphore chan struct{},
) {
	defer func() { <-semaphore }()
	defer conn.Close()

	semaphore <- struct{}{}
	logger := NewCli()
	scanner := bufio.NewScanner(conn)
	privateKey, publicKey, err := GenerateRSAKeyPair()
	if err != nil {
		fmt.Println("Error generating RSA key pair:", err)
		return
	}

	client := &client{
		serverPublicKey: publicKey,
		privateKey:      privateKey,
		logger:          NewCli(),
		conn:            conn,
	}

	client.negotiate()

	for scanner.Scan() {
		cmd := scanner.Text()

		logger.Log(fmt.Sprintf("[LOG]: %s", cmd))

		parts := strings.Fields(cmd)

		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "AUTH":
			if len(parts) != 3 {
				client.respond("ERR wrong number of arguments for AUTH")
				continue
			}

			incomingUserHash := sha256.Sum256([]byte(parts[1]))
			incomingPasswordHash := sha256.Sum256([]byte(parts[2]))
			incomingUserHashString := hex.EncodeToString(incomingUserHash[:])
			incomingPasswordHashString := hex.EncodeToString(incomingPasswordHash[:])

			if credentials.User != incomingUserHashString ||
				credentials.Password != incomingPasswordHashString {
				client.respond("ERR authentication faild")
				continue
			}

		case "SET":
			if len(parts) != 3 {
				client.respond("ERR wrong number of arguments for SET")
				continue
			}
			err := cache.Set(parts[1], parts[2])
			if err != nil {
				client.respond(fmt.Sprintf("ERR %s", err))
				continue
			}
			_, err = conn.Write([]byte("OK\n"))
			if err != nil {
				logger.ErrLog.Println("ERR couldn't write back to client")
			}

		case "GET":
			if len(parts) != 2 {
				client.respond("ERR wrong number of arguments for GET")
				continue
			}
			value, ok := cache.Get(parts[1])
			if ok {
				client.respond(fmt.Sprintf("VALUE %s", value))
			} else {
				client.respond("NOT_FOUND")
			}

		case "DELETE":
			if len(parts) != 2 {
				client.respond("ERR wrong number of arguments for DELETE")
				continue
			}
			cache.Delete(parts[1])
			client.respond("OK")

		default:
			client.respond("ERR unknown verb")
		}
	}
}

func backup(cache *cache) {
	cacheData, err := json.MarshalIndent(cache, "", "    ")
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

func readFromBackup(cache *cache, backupOn bool) error {
	if backupOn {
		fileData, err := os.ReadFile("backup.json")
		if err != nil {
			fmt.Println("Error reading JSON file: ", err)
			return err
		}
		err = json.Unmarshal(fileData, &cache)
		if err != nil {
			fmt.Println("Error unmarshalling JSON: ", err)
			return err
		}

		return nil
	}

	return fmt.Errorf("backup is off")
}

func StartServer(serverConfig *ServerConfig) {
	var cache *cache

	err := readFromBackup(cache, serverConfig.PoolConfig.BackupOn)
	if err != nil {
		cache = newCache(
			serverConfig.PoolConfig.NodeLimit,
			serverConfig.PoolConfig.CacheLimit,
			serverConfig.PoolConfig.TimeToLive,
			serverConfig.PoolConfig.NodeSize,
		)
	}

	cli := NewCli()
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverConfig.Port))
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %s", err))
		return
	}
	defer listener.Close()

	serverCredentials := &credentials{
		User:     serverConfig.User,
		Password: serverConfig.Password,
	}
	semaphore := make(chan struct{}, serverConfig.PoolConfig.MaxConns)

	if serverConfig.PoolConfig.BackupOn {
		interval := time.Duration(serverConfig.PoolConfig.BackupCycle) * time.Millisecond
		go Interval(interval, func() { backup(cache) })
	}

	cli.Launch(fmt.Sprintf("Lebre cache server initiated. Listening on port %d", serverConfig.Port))

	for {
		conn, err := listener.Accept()
		if err != nil {
			cli.Error(fmt.Sprintf("Error accepting connection: %s", err))
			continue
		}
		go handleClient(conn, cache, serverCredentials, semaphore)
	}
}
