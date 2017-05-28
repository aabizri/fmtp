package main

import (
	"context"
	"fmt"

	"github.com/abiosoft/ishell"
)

func disconnectCmd(c *ishell.Context) {
	if conn == nil {
		c.Err(fmt.Errorf("cannot disconnect as there's no conn currently active"))
		return
	}

	err := conn.Disconnect(context.Background())
	if err != nil {
		c.Err(err)
	}

	conn = nil
	c.Actions.SetPrompt(normalPrompt)
}
