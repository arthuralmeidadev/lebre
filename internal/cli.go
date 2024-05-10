package internal

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/howeyc/gopass"
)

type cli struct {
	launch  *color.Color
	warning *color.Color
	prompt  *color.Color
	errLog  *color.Color
	info    *color.Color
	log     *color.Color
}

func (cli *cli) Lebre() {
	cli.warning.Println(`			     ┌──┐                                                           
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
}

func (cli *cli) GreenHighlight() {

}

func (cli *cli) Highlight(text string) {
	cli.info.Println(text)

}

func (cli *cli) Error(text string) {
	cli.errLog.Println(text)
}

func (cli *cli) Fatal(err error) {
	cli.Error(fmt.Sprintf("%s", err))
	os.Exit(1)
}

func (cli *cli) Input(text string, storage any) {
	cli.prompt.Printf("%s: ", text)
	fmt.Scanf("%s\n", storage)
}

func (cli *cli) HiddenInput(text string) string {
	cli.prompt.Printf("%s: ", text)
	input, err := gopass.GetPasswdMasked()
	if err != nil {
		cli.Fatal(err)
	}
	return string(input)
}

func (cli *cli) Log(text string) {
	cli.log.Println(text)
}

func (cli *cli) Launch(text string) {
	cli.launch.Println(text)
}

func (cli *cli) Help() {
	bold := color.New(color.Bold)
	yellow := color.New(color.FgYellow)
	commandsTable := [][3]string{
		{"init", "", "Creates a new server"},
		{"start", "", "Starts the server"},
		{"status", "", "Returns the status of the server"},
		{"config (get|set)", "", "Server configuration"},
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

func NewCli() *cli {
	return &cli{
		launch:  color.New(color.FgGreen).Add(color.Bold),
		warning: color.New(color.FgYellow),
		prompt:  color.New(color.FgCyan),
		errLog:  color.New(color.FgHiRed),
		info:    color.New(color.Bold),
		log:     color.New(color.FgHiBlack),
	}
}
