package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/howeyc/gopass"
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
	logger := color.New(color.FgHiBlack)
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		cmd := scanner.Text()
		logger.Println("[LOG]: ", cmd)

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
		{"init", "", "Creates a new server"},
		{"start", "", "Starts the server"},
		{"config (get|set)", "", "Configures the server"},
		{"help", "[command]", "Shows this menu"},
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
	prompt := color.New(color.FgCyan)
	warning := color.New(color.FgYellow)
	info := color.New(color.Bold)
	errLog := color.New(color.FgHiRed)
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

	case "init":
		type ServerConfig struct {
			name             string
			user             string
			password         string
			port             uint32
			maxConns         uint8
			timeoutThreshold uint16
			backUpOn         string
			backUpCycle      uint32
			nodeLimit        uint32
			cacheLimit       uint32
		}

		serverConfig := ServerConfig{
			port:             5051,
			maxConns:         15,
			timeoutThreshold: 5000,
			backUpCycle:      300000,
			nodeLimit:        3500,
			cacheLimit:       5242880,
		}

		var passwordRepeat string

		warning.Println(`			     ┌──┐                                                           
                             │  └─┐                                                         
                            ┌┘    │                                                         
                  ┌──────┐  │     └┐                                                        
                  │      └┐ │      │                                                        
                  │       └─┘      │                                                        
                  └─┐        ─┐    │                                                        
                    └─┐       └─   ├──┐                                                     
                      └─┐          │  └───┐                                                 
                        └─┐               └┐                                                
                          ├┐               └┐                                               
                         ┌┴┴─        ─┬─┐   └┐                                              
                     ┌───┘            └─┘    └┐                                             
                  ┌──┘                        │                                             
               ┌──┘               │          ┌┘                                             
             ┌─┘                  └┬──┐    ┌─┘                                              
           ┌─┘                     │  └────┘                                                
         ┌─┘                       │  ___       _______   ________  ________  _______       
        ┌┘                         │ |\  \     |\  ___ \ |\   __  \|\   __  \|\  ___ \      
       ┌┘                  │      │  \ \  \    \ \   __/|\ \  \|\ /\ \  \|\  \ \   __/|     
      ┌┘            ┌────┐ │      │   \ \  \    \ \  \_|/_\ \   __  \ \   _  _\ \  \_|/__   
      │          ┌──┘    ├┐      │     \ \  \____\ \  \_|\ \ \  \|\  \ \  \\  \\ \  \_|\ \  
     ┌┘         ─┘       └┼┐     │      \ \_______\ \_______\ \_______\ \__\\ _\\ \_______\ 
     │                    ││     │       \|_______|\|_______|\|_______|\|__|\|__|\|_______| 
┌────┘                   ┌┴┴┐    │                                                          
│                       ┌┤  │    │          ####   ## # ##  #  #  # # ## ####  ##  #        
└┐     ┌─┬┬────        ─┴┴┐ │    └─┐ ####### #### # # ##########   ##  #  # #  ##### ## ### 
 └─────┘ ││               └─┤      │#### #   ### ## ## #   #####     #    # #               
         └┤                 └──────      # # ##   ##  #  ###### #                           
          └─────────────────#  #    ############    ##   #  ##                              
          # #   #    #    #   #    # ###  # # #                                             `)

		info.Println("\n Lebre cache server v1.0 running init")
		prompt.Print("\nServer name: ")
		fmt.Scanf("%s\n", &serverConfig.name)
		prompt.Print("User: ")
		fmt.Scanf("%s\n", &serverConfig.user)

		prompt.Print("Password: ")
		pwdInput, err := gopass.GetPasswdMasked()
		if err != nil {
			fmt.Println("Error reading password:", err)
			return
		}
		serverConfig.password = string(pwdInput)

		prompt.Print("Repeat password: ")
		rPwdInput, err := gopass.GetPasswdMasked()
		if err != nil {
			fmt.Println("Error reading password:", err)
			return
		}
		passwordRepeat = string(rPwdInput)

		for serverConfig.password != passwordRepeat ||
			len(serverConfig.password) < 8 {

			if serverConfig.password != passwordRepeat {
				errLog.Println("Passwords do not match")
			}

			if len(serverConfig.password) < 8 {
				errLog.Println("Password too short")
			}

			prompt.Print("Password: ")
			pwdInput, err := gopass.GetPasswdMasked()
			if err != nil {
				errLog.Println("Error reading password:", err)
				return
			}
			serverConfig.password = string(pwdInput)

			prompt.Print("Repeat password: ")
			rPwdInput, err := gopass.GetPasswdMasked()
			if err != nil {
				errLog.Println("Error reading password:", err)
				return
			}
			passwordRepeat = string(rPwdInput)
		}

		prompt.Printf("Port (DEFAULT %d): ", serverConfig.port)
		fmt.Scanf("%s\n", &serverConfig.port)
		prompt.Printf("Maximum number of connections (DEFAULT %d): ", serverConfig.maxConns)
		fmt.Scanf("%d\n", &serverConfig.maxConns)
		prompt.Printf("Timout threshold in milliseconds (DEFAULT %d): ", serverConfig.timeoutThreshold)
		fmt.Scanf("%d\n", &serverConfig.timeoutThreshold)
		prompt.Print("Turn on backup? (y(yes)/n(no)): ")
		fmt.Scanf("%s\n", &serverConfig.backUpOn)
		if serverConfig.backUpOn == "y" {
			prompt.Printf("Backup cycle in milliseconds (DEFAULT %d): ", serverConfig.backUpCycle)
			fmt.Scanf("%d\n", &serverConfig.backUpCycle)
		}
		prompt.Printf("Limit for simultaneous nodes (DEFAULT %d): ", serverConfig.nodeLimit)
		fmt.Scanf("%d\n", &serverConfig.nodeLimit)
		prompt.Printf("Cache limit in bytes (DEFAULT %d): ", serverConfig.cacheLimit)
		fmt.Scanf("%d\n", &serverConfig.cacheLimit)
		return

	case "start":
		if len(arguments) == 3 {
			if arguments[1] == "--config" || arguments[1] == "-c" {
				fmt.Print("")
			} else {
				errLog.Printf("Invalid argument or command part '%s'\n", arguments[1])
				os.Exit(1)
			}
		}

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
				errLog.Println("Error accepting connection:", err)
				continue
			}
			go handleClient(conn, cache)
		}

	case "config":
		if len(arguments) != 3 {
			errLog.Println("Missing arguments for 'config'\ntype 'lebre help' to see all available commands")
			os.Exit(1)
		}

	default:
		errLog.Println("Error: unknown command\ntype 'lebre help' to see all available commands")
		os.Exit(1)
	}
}
