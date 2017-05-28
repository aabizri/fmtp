package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime/pprof"

	"github.com/aabizri/fmtp"
	"github.com/urfave/cli"
)

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "local",
		Value: "localID",
		Usage: "ID for the local client",
	},
	cli.StringFlag{
		Name:  "cpu",
		Usage: "Where to write the cpu profile",
	},
	cli.StringFlag{
		Name:  "addr",
		Value: "127.0.0.1:8050",
		Usage: "address to listen to",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "fmtp"
	app.Usage = "Listen to incoming FMTP messages"
	app.Flags = flags
	app.Action = action

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func action(c *cli.Context) error {
	if cpu := c.String("cpu"); cpu != "" {
		f, err := os.Create(cpu)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Create the client
	local := fmtp.ID(c.String("local"))
	client, err := fmtp.NewClient(local)
	if err != nil {
		return err
	}

	// Create the address
	addr := c.String("addr")

	// Create the main handler
	handler := fmtp.HandlerFunc(func(conn *fmtp.Conn, msg *fmtp.Message) {
		fmt.Printf("Server> received new %s message:\n", msg.Typ().String())
		if msg.Typ() == fmtp.Operator {
			io.Copy(os.Stdout, msg.Body)
			fmt.Println()
		}
	})

	// Create the server and its handlers
	srv := client.NewServer(addr, handler)
	srv.AcceptTCP = func(addr net.Addr) bool {
		fmt.Printf("Server> received new TCP connection from %s\n", addr)
		return true
	}
	srv.NotifyConn = func(addr net.Addr, rem fmtp.ID) {
		fmt.Printf("Server> connection established with %s (%s)", rem, addr)
	}
	return srv.ListenAndServe()
}
