package internal

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
)

type PoolConfig struct {
	MaxConns         uint8  `json:"MaxConns"`
	TimeoutThreshold uint16 `json:"TimeoutThreshold"`
	BackupOn         bool   `json:"BackUpOn"`
	BackupCycle      uint32 `json:"BackUpCycle"`
	TimeToLive       uint16 `json:"TimeToLive"`
	NodeLimit        uint32 `json:"NodeLimit"`
	CacheLimit       uint32 `json:"CacheLimit"`
	IdleThreshold    uint16 `json:"IdleThreshold"`
}

type ServerConfig struct {
	Name             string      `json:"Name"`
	User             string      `json:"User"`
	Password         string      `json:"Password"`
	Port             uint32      `json:"Port"`
	EnableEncryption bool        `json:"EnableEncryption"`
	PoolConfig       *PoolConfig `json:"PoolConfig"`
}

type Cache struct {
	data  map[string]string
	mutex sync.RWMutex
}

type credentials struct {
	User     string
	Password string
}

func newCache() *Cache {
	return &Cache{
		data: make(map[string]string),
	}
}

func (cache *Cache) Set(key, value string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.data[key] = value
}

func (cache *Cache) Get(key string) (string, bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	value, ok := cache.data[key]
	return value, ok
}

func (cache *Cache) Delete(key string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	delete(cache.data, key)
}

func handleClient(conn net.Conn, cache *Cache, credentials *credentials) {
	defer conn.Close()

	logger := NewCli()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		cmd := scanner.Text()
		logger.Log(fmt.Sprintf("[LOG]: %s", cmd))

		parts := strings.Fields(cmd)

		if len(parts) == 0 {
			continue
		}

		// VERB KEY VALUE?
		switch parts[0] {
		case "AUTH":
			if len(parts) != 3 {
				conn.Write([]byte("ERR Wrong number of arguments for AUTH\n"))
				continue
			}

			incomingUserHash := sha256.Sum256([]byte(parts[1]))
			incomingPasswordHash := sha256.Sum256([]byte(parts[2]))
			incomingUserHashString := hex.EncodeToString(incomingUserHash[:])
			incomingPasswordHashString := hex.EncodeToString(incomingPasswordHash[:])

			if credentials.User != incomingUserHashString ||
				credentials.Password != incomingPasswordHashString {
				conn.Write([]byte("ERR Authentication faild\n"))
				continue
			}

		case "SET":
			if len(parts) != 3 {
				conn.Write([]byte("ERR Wrong number of arguments for SET\n"))
				continue
			}
			cache.Set(parts[1], parts[2])
			conn.Write([]byte("OK\n"))

		case "GET":
			if len(parts) != 2 {
				conn.Write([]byte("ERR Wrong number of arguments for GET\n"))
				continue
			}
			value, ok := cache.Get(parts[1])
			if ok {
				conn.Write([]byte(fmt.Sprintf("VALUE %s\n", value)))
			} else {
				conn.Write([]byte("NOT_FOUND\n"))
			}

		case "DELETE":
			if len(parts) != 2 {
				conn.Write([]byte("ERR Wrong number of arguments for DELETE\n"))
				continue
			}
			cache.Delete(parts[1])
			conn.Write([]byte("OK\n"))

		default:
			conn.Write([]byte("ERR Unknown command\n"))
		}
	}
}

func StartServer(serverConfig *ServerConfig) {
	cache := newCache()
	cli := NewCli()
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverConfig.Port))
	if err != nil {
		cli.Error(fmt.Sprintf("Error: %s", err))
		return
	}
	defer listener.Close()

	cli.Launch(fmt.Sprintf("Lebre cache server initiated. Listening on port %d", serverConfig.Port))

	serverCredentials := &credentials{
		User:     serverConfig.User,
		Password: serverConfig.Password,
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			cli.Error(fmt.Sprintf("Error accepting connection: %s", err))
			continue
		}
		go handleClient(conn, cache, serverCredentials)
	}
}
