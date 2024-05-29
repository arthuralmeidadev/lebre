# Lebre Cache Server 🐇

```
                             ┌──┐                                                          
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
          # #   #    #    #   #    # ###  # # #                                            
```

## Description

Lebre is a caching database server built in Go designed to be fast and simple.

You can download the Javascript API for Lebre through the node package manager:
```console
node i lebre
```
or using Yarn:
```console
yarn add lebre
```
Available at: https://www.npmjs.com/package/lebre

These are the available CLI commands

```
┌───────────────────────────────────────────────────────────────────────────────────────┐
│       init                                :   Creates a new server                    │
│       start                               :   Starts the server                       │
│       help                 [command]      :   Shows this menu                         │
└───────────────────────────────────────────────────────────────────────────────────────┘
```
After running ```lebre init``` this is the resulting config file:

### config.json example
```json
{
    "user": "f0bdfe12c09d3...",
    "password": "11aec05dc27...",
    "port": 5051,
    "enableEncryption": true,
    "poolConfig": {
        "maxConns": 15,
        "ConnectionTimeout": 5000,
        "backUpOn": false,
        "backUpCycle": 300000,
        "timeToLive": 300,
        "nodeLimit": 3500,
        "nodeSize": 1024,
        "cacheLimit": 5242880,
        "idleThreshold": 3600
    }
}
```

## The protocol

The protocol used to transfer messages over TCP is built on the simple idea of boundaries by length.
Messages will be written to buffers with their length prepended in a 32-bit long integer value stored in Big Endian order.
It consists of 4 parts:

Version - V*.*

Verb - AUTH/SET/GET/DELETE

Key - {string, no spaces}

Value (AUTH/SET) - {string/hash, spaces as '\\u0020'}

### Examples

```
V1.0 AUTH root password1234
```
```
V1.0 SET ExampleValue Hello\\u0020World!
```
```
V1.0 GET ExampleValue
```
```
V1.0 DELETE ExampleValue
```

### Encryption

The protocol utilizes RSA asymmetric encryption to encrypt the exchange of data between client and server.
To achieve this, at the establishment of a connection, the client and server will generate each a pair of asymetric keys and then exchange the public keys.
