package main

import (
	"fmt"
	"strings"

	"github.com/aabizri/fmtp"
)

func parseAddress(addr string) (id fmtp.ID, host string, err error) {
	splits := strings.Split(addr, "@")
	if len(splits) != 2 {
		err = fmt.Errorf("invalid address format")
		return
	}

	id = fmtp.ID(splits[0])
	host = splits[1]
	return
}
