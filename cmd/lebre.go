package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
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
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		cmd := scanner.Text()
		fmt.Println("LOG: ", cmd)

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

func help() {
	bold := color.New(color.Bold)
	yellow := color.New(color.FgYellow)
	commandsTable := [][3]string{
		[3]string{"init", "", "Creates a new server"},
		[3]string{"start", "", "Starts the server"},
		[3]string{"config (get|set)", "", "Configures the server"},
		[3]string{"help", "[command]", "Shows this menu"},
	}

	title := color.New(color.FgHiBlue).Add(color.Bold)
	title.Println("\nLebre cache server help:")
	fmt.Println("┌───────────────────────────────────────────────────────────────────────────────────────┐")
	for _, row := range commandsTable {
		bold.Printf("│\t%-20s", row[0])
		yellow.Printf(" %-15s", row[1])
		fmt.Printf(":\t%-40s│\n", row[2])
	}
	fmt.Println("└───────────────────────────────────────────────────────────────────────────────────────┘")
}

func main() {
	arguments := os.Args[1:]
	cache := NewCache()

	if len(arguments) == 0 {
		help()
		return
	}

	switch arguments[0] {
	case "help":
		help()
		return

	case "start":
		listener, err := net.Listen("tcp", ":5052")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer listener.Close()

		initMsg := color.New(color.FgGreen).Add(color.Bold)
		initMsg.Println("Lebre cache server initiated. Listening on port 5052")

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			go handleClient(conn, cache)
		}

	case "config":
		if len(arguments) != 3 {
			log.Fatal("Missing arguments for 'config'\ntype 'lebre help' to see all available commands")
		}

	default:
		log.Fatal("Error: unknown command\ntype 'lebre help' to see all available commands")
	}
}
