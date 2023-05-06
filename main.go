package main

import (
	"flag"
	"fmt"
	"os"
)

type Address struct {
	host string
	port int
}

func (a *Address) String() string {
	return fmt.Sprintf("%s:%d", a.host, a.port)
}

func main() {
	src := flag.String("src", ":8989", "listen host:port")
	dst := flag.String("dst", "8.8.8.8:53", "destination host:port")
	enableUDP := flag.Bool("udp", false, "enable udp ( experimental, needs to enabled at both sides )")
	transport := flag.String("transport", "tcp", "[ tcp, websocket ]")
	host := flag.String("host-header", "", "use it at the relay node to specify the host header in websocket transport handshake")
	wsPath := flag.String("ws-path", "/", "route for ws transport, useful when gate is behind reverse proxy, /ws for example")
	serverType := flag.Int("type", 0, "1: gate, 2: relay")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help == true {
		flag.PrintDefaults()
		os.Exit(0)
	}

	switch *transport {
	case "tcp", "websocket":
		{
			break
		}
	default:
		{
			println("wrong transport type")
			os.Exit(1)
		}
	}

	switch *serverType {
	case 1, 2:
		{
			wsInfo := ""

			if *transport == "websocket" {
				wsInfo = `wsPath: ` + *wsPath
			}
			println(fmt.Sprintf(`
transport: %s
listen on: %s
destination: %s
`+wsInfo, *transport, *src, *dst))
			println(`udp-enabled :`, *enableUDP, "\n")
		}
		break

	default:
		println("invalid serverType", *serverType)
		flag.PrintDefaults()
		os.Exit(1)
	}

	tr := "<--->"
	if *transport == "websocket" {
		tr = "<-ws->"
	}

	var relay IService
	relay = NewTCPRelay(*src, *dst, *enableUDP, *transport, *host, *wsPath, *serverType)
	println(fmt.Sprintf("%s %s %s", *src, tr, *dst))
	relay.Start()
}
