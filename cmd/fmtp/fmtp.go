package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aabizri/fmtp"
	"github.com/abiosoft/ishell"
)

const (
	prompt       = "fmtp"
	normalPrompt = prompt + "> "
)

// interactive commands
var icommands = []*ishell.Cmd{
	&ishell.Cmd{
		Name: "send",
		Func: sendCmd,
		Help: "sends a message to the remote",
	},
	&ishell.Cmd{
		Name: "connect",
		Func: connectCmd,
		Help: "connect to the remote",
	},
	&ishell.Cmd{
		Name: "dial",
		Func: dialCmd,
		Help: "connect & associate to the remote",
	},
	&ishell.Cmd{
		Name: "disconnect",
		Func: disconnectCmd,
		Help: "disconnect from the remote",
	},
	&ishell.Cmd{
		Name: "associate",
		Func: associateCmd,
		Help: "associate to the remote",
	},
	&ishell.Cmd{
		Name: "deassociate",
		Func: deassociateCmd,
		Help: "deassociate from the remote",
	},
}

var (
	client *fmtp.Client
	conn   *fmtp.Conn
)

func main() {

	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()
	shell.Actions.SetPrompt(normalPrompt)

	// register the commands
	for _, cmd := range icommands {
		shell.AddCmd(cmd)
	}

	// Extract the address
	if len(os.Args) >= 2 {
		// Extract the address
		address := os.Args[1]

		// Split it
		id, addr, err := parseAddress(address)
		if err != nil {
			shell.Actions.Printf("error: %v", err)
			return
		}

		// Create client
		client, err := fmtp.NewClient("localID")
		if err != nil {
			shell.Actions.Printf("error: %v", err)
			return
		}

		// Dial
		conn, err = client.Dial(context.Background(), addr, id)
		if err != nil {
			shell.Actions.Printf("error: %v", err)
			return
		}

		// Set the correct prompt
		shell.Actions.SetPrompt(fmt.Sprintf("%s (%s)> ", prompt, address))
	}
	if len(os.Args) >= 3 {
		fmt.Println(os.Args)
		shell.Process(os.Args[2:]...)
		fmt.Println("non-interactive mode")
	} else {
		interactive(shell)
	}
}

func interactive(sh *ishell.Shell) {
	// display welcome info.
	sh.Println("FMTP interactive shell")

	// run shell
	sh.Run()

	// close
	sh.Close()
}
