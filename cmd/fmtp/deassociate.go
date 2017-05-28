package main

import (
	"context"
	"fmt"

	"github.com/abiosoft/ishell"
)

func deassociateCmd(c *ishell.Context) {
	if conn == nil {
		c.Err(fmt.Errorf("cannot deassociate as there's no conn currently active"))
		return
	}

	err := conn.Deassociate(context.Background())
	if err != nil {
		c.Err(err)
	}
}
