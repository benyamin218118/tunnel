# tunnel

```
$ tunnel -h

  -dst string
        destination host:port (default "8.8.8.8:53")
  -h    help
  -host-header string
        use it at the relay node to specify the host header in websocket transport handshake
  -src string
        listen host:port (default ":8989")
  -transport string
        [ tcp, websocket ] (default "tcp")
  -type int
        1: gate, 2: relay
  -udp
        enable udp ( experimental, needs to enabled at both sides )
  -ws-path string
        route for ws transport, useful when gate is behind reverse proxy, /ws for example (default "/")

```