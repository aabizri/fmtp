package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aabizri/fmtp"
	"github.com/abiosoft/ishell"
)

func sendCmd(c *ishell.Context) {
	// Get the connection
	if conn == nil {
		c.Err(fmt.Errorf("cannot send message when no connection has been created"))
		return
	}

	// Create context
	ctx := context.Background()

	// Create message
	txt := strings.Join(c.Args, " ")
	msg, err := fmtp.NewOperatorMessageString(txt)
	if err != nil {
		c.Err(fmt.Errorf("couldn't create new message: %v", err))
		return
	}

	// Send the message
	err = conn.Send(ctx, msg)
	if err != nil {
		c.Err(fmt.Errorf("couldn't send message: %v", err))
	}

	return
}
