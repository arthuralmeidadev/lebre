package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lebre/internal"
	"os"
)

func main() {
	cli := internal.NewCli()
	arguments := os.Args[1:]

	if len(arguments) == 0 {
		cli.Help("")
		return
	}

	switch arguments[0] {
	case "help":
		if len(arguments) != 2 {
			cli.Help("")
		} else {
			cli.Help(arguments[1])
		}
		return

	case "init":
		serverConfig := internal.DefaultServerConfig()
		var passwordRepeat string

		cli.Lebre()
		cli.Highlight("\n Lebre cache server v1.0 running init\n")
		cli.Input("Server name", &serverConfig.Name)
		cli.Input("User", &serverConfig.User)
		serverConfig.Password = cli.HiddenInput("Password")
		passwordRepeat = cli.HiddenInput("Repeat password")

		for serverConfig.Password != passwordRepeat ||
			len(serverConfig.Password) < 8 {

			if serverConfig.Password != passwordRepeat {
				cli.Error("Passwords do not match")
			}

			if len(serverConfig.Password) < 8 {
				cli.Error("Password too short")
			}

			serverConfig.Password = cli.HiddenInput("Password")
			passwordRepeat = cli.HiddenInput("Repeat password")
		}

		cli.Input(
			fmt.Sprintf("Port (DEFAULT %d)", serverConfig.Port),
			&serverConfig.Port,
		)

		var enableEncryption string
		cli.Input("Enable encryption (RECOMENDED: YES)? (y(yes) / n(no))", &enableEncryption)
		if enableEncryption == "n" {
			serverConfig.EnableEncryption = false
		}

		cli.Input(
			fmt.Sprintf("Maximum number of simultaneous connections (DEFAULT %d)", serverConfig.PoolConfig.MaxConns),
			&serverConfig.PoolConfig.MaxConns,
		)
		cli.Input(
			fmt.Sprintf("Connection timout threshold in milliseconds (DEFAULT %d)", serverConfig.PoolConfig.TimeoutThreshold),
			serverConfig.PoolConfig.TimeoutThreshold,
		)

		var backupOn string
		cli.Input("Turn on backup? (y(yes) / n(no))", &backupOn)
		if backupOn == "y" {
			serverConfig.PoolConfig.BackupOn = true
			cli.Input(
				fmt.Sprintf("Backup cycle in milliseconds (DEFAULT %d)", serverConfig.PoolConfig.BackupCycle),
				serverConfig.PoolConfig.BackupCycle,
			)
		} else {
			serverConfig.PoolConfig.BackupOn = false
		}

		cli.Input(
			fmt.Sprintf("Cached value lifetime in milliseconds (DEFAULT %d)", serverConfig.PoolConfig.TimeToLive),
			serverConfig.PoolConfig.TimeToLive,
		)
		cli.Input(
			fmt.Sprintf("Limit for simultaneous nodes (DEFAULT %d)", serverConfig.PoolConfig.NodeLimit),
			serverConfig.PoolConfig.NodeLimit,
		)
		cli.Input(
			fmt.Sprintf("single cache node size maximum limit in bytes (DEFAULT %d)", serverConfig.PoolConfig.NodeSize),
			serverConfig.PoolConfig.NodeSize,
		)
		cli.Input(
			fmt.Sprintf("Cache limit in bytes (DEFAULT %d)", serverConfig.PoolConfig.CacheLimit),
			serverConfig.PoolConfig.CacheLimit,
		)
		cli.Input(
			fmt.Sprintf("Maximum idle time in seconds before memory cleanup (DEFAULT %d)", serverConfig.PoolConfig.IdleThreshold),
			serverConfig.PoolConfig.IdleThreshold,
		)

		userHash := sha256.Sum256([]byte(serverConfig.User))
		serverConfig.User = hex.EncodeToString(userHash[:])

		passwordHash := sha256.Sum256([]byte(serverConfig.Password))
		serverConfig.Password = hex.EncodeToString(passwordHash[:])

		serverConfigJsonData, err := json.MarshalIndent(serverConfig, "", "    ")
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}

		err = os.WriteFile("config.json", serverConfigJsonData, 0644)
		if err != nil {
			fmt.Println("Error writing JSON to file: ", err)
			return
		}

		return

	case "start":
		server := &internal.LebreServer{}

		if len(arguments) == 3 {
			if arguments[1] == "--config" || arguments[1] == "-c" {
				fileData, err := os.ReadFile(arguments[2])
				if err != nil {
					fmt.Println("Error reading JSON file: ", err)
					return
				}
				err = json.Unmarshal(fileData, &server.ServerConfig)
				if err != nil {
					fmt.Println("Error unmarshalling JSON: ", err)
					return
				}

				server.Start()

			} else {
				cli.Error(fmt.Sprintf("Invalid argument or command part '%s'", arguments[1]))
				os.Exit(1)
			}
		}

		fileData, err := os.ReadFile("config.json")
		if err != nil {
			fmt.Println("Error reading JSON file: ", err)
			return
		}

		err = json.Unmarshal(fileData, &server.ServerConfig)
		if err != nil {
			fmt.Println("Error unmarshalling JSON: ", err)
			return
		}

		server.Start()

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

	select {}
}
