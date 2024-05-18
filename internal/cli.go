package internal

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/howeyc/gopass"
)

type Cli struct {
	CmdTitle   *color.Color
	LaunchText *color.Color
	Warning    *color.Color
	Prompt     *color.Color
	ErrLog     *color.Color
	Info       *color.Color
	LogText    *color.Color
}

func (cli *Cli) Lebre() {
	cli.Warning.Println(`			     ┌──┐                                                           
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

func (cli *Cli) GreenHighlight() {

}

func (cli *Cli) Highlight(text string) {
	cli.Info.Println(text)

}

func (cli *Cli) Error(text string) {
	cli.ErrLog.Println(text)
}

func (cli *Cli) Fatal(err error) {
	cli.Error(fmt.Sprintf("%s", err))
	os.Exit(1)
}

func (cli *Cli) Input(text string, storage any) {
	cli.Prompt.Printf("%s: ", text)
	fmt.Scanf("%s\n", storage)
}

func (cli *Cli) HiddenInput(text string) string {
	cli.Prompt.Printf("%s: ", text)
	input, err := gopass.GetPasswdMasked()
	if err != nil {
		cli.Fatal(err)
	}
	return string(input)
}

func (cli *Cli) Log(text string) {
	cli.LogText.Println(text)
}

func (cli *Cli) Launch(text string) {
	cli.LaunchText.Println(text)
}

func (cli *Cli) Help(command string) {
	cli.CmdTitle.Println("\nLebre cache server help:")

	switch command {
	case "init":
		fmt.Print("\n│ ")
		cli.Info.Print(command)
		fmt.Print(" (")
		cli.Warning.Print("--default")
		fmt.Print(" | ")
		cli.Warning.Print("-d")
		fmt.Println(")")
		fmt.Println("│ This command initializes a new Lebre instance.")
		fmt.Print("│ When run, if the flag ")
		cli.Warning.Print("--default")
		fmt.Println(" is not set, the user will be prompted with the desired settings.")
		fmt.Println()

	case "start":
		fmt.Print("\n│ ")
		cli.Info.Print(command)
		cli.Warning.Print(" --config")
		fmt.Println(" [configuration file path (*.lebre)]")
		fmt.Println("│ This command starts the Lebre server")
		fmt.Print("│ The flag ")
		cli.Warning.Print("--config")
		fmt.Println(" can be used to provide a configuration file path,")
		fmt.Println("│ which will override the default configuration from the server setup")
		fmt.Println()

	default:
		commandsTable := [][3]string{
			{"init", "", "Creates a new server"},
			{"start", "", "Starts the server"},
			{"status", "", "Returns the status of the server"},
			{"config (get|set)", "", "Server configuration"},
			{"help", "[command]", "Shows this menu"},
		}

		fmt.Println("┌───────────────────────────────────────────────────────────────────────────────────────┐")
		for _, row := range commandsTable {
			cli.Info.Printf("│\t%-20s", row[0])
			cli.Warning.Printf(" %-15s", row[1])
			fmt.Printf(":\t%-40s│\n", row[2])
		}
		fmt.Println("└───────────────────────────────────────────────────────────────────────────────────────┘")
	}

}

func NewCli() *Cli {
	return &Cli{
		CmdTitle:   color.New(color.FgHiBlue).Add(color.Bold),
		LaunchText: color.New(color.FgGreen).Add(color.Bold),
		Warning:    color.New(color.FgYellow),
		Prompt:     color.New(color.FgCyan),
		ErrLog:     color.New(color.FgHiRed),
		Info:       color.New(color.Bold),
		LogText:    color.New(color.FgHiBlack),
	}
}
