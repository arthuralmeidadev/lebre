package main

import (
	"bufio"
	"fmt"
	"lebre/internal"
	"net"
	"os"
	"strings"
	"sync"
)

type Cache struct {
	data  map[string]string
	mutex sync.RWMutex
}

func NewCache() *Cache {
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

func handleClient(conn net.Conn, cache *Cache) {
	logger := internal.NewCli()
	defer conn.Close()
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

func main() {
	cli := internal.NewCli()
	arguments := os.Args[1:]
	cache := NewCache()

	if len(arguments) == 0 {
		cli.Help()
		return
	}

	switch arguments[0] {
	case "help":
		cli.Help()
		return

	case "init":
		type PoolConfig struct {
			maxConns         uint8
			timeoutThreshold uint16
			backUpOn         string
			backUpCycle      uint32
			nodeLimit        uint32
			cacheLimit       uint32
			idleThreshold    uint16
		}

		type ServerConfig struct {
			name       string
			user       string
			password   string
			port       uint32
			poolConfig PoolConfig
		}

		serverConfig := ServerConfig{
			port: 5051,
			poolConfig: PoolConfig{
				maxConns:         15,
				timeoutThreshold: 5000,
				backUpCycle:      300000,
				nodeLimit:        3500,
				cacheLimit:       5242880,
				idleThreshold:    3600,
			},
		}

		var passwordRepeat string

		cli.Lebre()
		cli.Highlight("\n Lebre cache server v1.0 running init\n")
		cli.Input("Server name", &serverConfig.name)
		cli.Input("User", &serverConfig.user)
		serverConfig.password = cli.HiddenInput("Password")
		passwordRepeat = cli.HiddenInput("Repeat password")

		for serverConfig.password != passwordRepeat ||
			len(serverConfig.password) < 8 {

			if serverConfig.password != passwordRepeat {
				cli.Error("Passwords do not match")
			}

			if len(serverConfig.password) < 8 {
				cli.Error("Password too short")
			}

			serverConfig.password = cli.HiddenInput("Password")
			passwordRepeat = cli.HiddenInput("Repeat password")
		}

		cli.Input(
			fmt.Sprintf("Port (DEFAULT %d)", serverConfig.port),
			&serverConfig.port,
		)
		cli.Input(
			fmt.Sprintf("Maximum number of connections (DEFAULT %d)", serverConfig.poolConfig.maxConns),
			&serverConfig.poolConfig.maxConns,
		)
		cli.Input(
			fmt.Sprintf("Timout threshold in milliseconds (DEFAULT %d)", serverConfig.poolConfig.timeoutThreshold),
			serverConfig.poolConfig.timeoutThreshold,
		)

		cli.Input("Turn on backup? (y(yes) / n(no))", &serverConfig.poolConfig.backUpOn)
		if serverConfig.poolConfig.backUpOn == "y" {
			cli.Input(
				fmt.Sprintf("Backup cycle in milliseconds (DEFAULT %d)", serverConfig.poolConfig.backUpCycle),
				serverConfig.poolConfig.backUpCycle,
			)
		}

		cli.Input(
			fmt.Sprintf("Limit for simultaneous nodes (DEFAULT %d)", serverConfig.poolConfig.nodeLimit),
			serverConfig.poolConfig.nodeLimit,
		)
		cli.Input(
			fmt.Sprintf("Cache limit in bytes (DEFAULT %d)", serverConfig.poolConfig.cacheLimit),
			serverConfig.poolConfig.cacheLimit,
		)
		cli.Input(
			fmt.Sprintf("Maximum idle time until memory cleanup in seconds (DEFAULT %d)", serverConfig.poolConfig.idleThreshold),
			serverConfig.poolConfig.idleThreshold,
		)

		return

	case "start":
		if len(arguments) == 3 {
			if arguments[1] == "--config" || arguments[1] == "-c" {
				fmt.Print("")
			} else {
				cli.Error(fmt.Sprintf("Invalid argument or command part '%s'", arguments[1]))
				os.Exit(1)
			}
		}

		listener, err := net.Listen("tcp", ":5052")
		if err != nil {
			cli.Error(fmt.Sprintf("Error: %s", err))
			return
		}
		defer listener.Close()

		cli.Launch("Lebre cache server initiated. Listening on port 5052")

		for {
			conn, err := listener.Accept()
			if err != nil {
				cli.Error(fmt.Sprintf("Error accepting connection: %s", err))
				continue
			}
			go handleClient(conn, cache)
		}

	case "config":
		if len(arguments) != 3 {
			cli.Error("Missing arguments for 'config'\ntype 'lebre help' to see all available commands")
			os.Exit(1)
		}

	default:
		cli.Error(fmt.Sprintf("Error: unknown command '%s'\n", arguments[0]))
		cli.Highlight("Type 'lebre help' to see all available commands")
		os.Exit(1)
	}
}
