package main

import (
	"context"
	"fmt"

	"github.com/abiosoft/ishell"
)

func associateCmd(c *ishell.Context) {
	if conn == nil {
		c.Err(fmt.Errorf("cannot send message when no connection has been created"))
		return
	}

	err := conn.Associate(context.Background())
	if err != nil {
		c.Err(err)
	}
}
