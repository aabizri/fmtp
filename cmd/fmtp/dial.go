package main

import (
	"context"
	"fmt"

	"github.com/aabizri/fmtp"
	"github.com/abiosoft/ishell"
)

func dialCmd(c *ishell.Context) {
	// Get the connection if there is one
	if conn != nil {
		c.Err(fmt.Errorf("cannot connect: there's already an active connection"))
		return
	}

	// Get the first argument
	if len(c.Args) == 0 {
		c.Err(fmt.Errorf("at least one argument is necessary: the address (ID@host)"))
		return
	}

	// Split it
	id, addr, err := parseAddress(c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	// Create client
	client, err := fmtp.NewClient("localID")
	if err != nil {
		c.Err(err)
		return
	}

	// Connect
	conn, err = client.Dial(context.Background(), addr, id)
	if err != nil {
		c.Err(err)
		return
	}

	c.Actions.Println("dial successful !")

	// Set the correct prompt
	c.Actions.SetPrompt(fmt.Sprintf("%s (%s)> ", prompt, c.Args[0]))
	return
}
