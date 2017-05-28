# Package fmtp implements support for Flight Message Transfer Protocol v2.0

## Usage

### Create a client
`client,_ := fmtp.NewClient("my id")`

### Connect & associate with a remote endpoint
`conn, _ := client.Dial("address","id")`

### Send a message
`conn.SendOperatorString("hello there")`

## TODO

- Test server-side (handlers,etc.)
- More tests
- Better logging
- More callbacks