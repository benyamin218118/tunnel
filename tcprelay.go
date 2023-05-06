package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type TCPRelay struct {
	listener      net.Listener
	udpEnabled    bool
	udpListener   *net.UDPConn
	connKeeper    *ConnKeeper
	udpBufferSize int
	src           string
	destination   string
	transport     string
	hostHeader    string
	wsPath        string
	serverType    int
}

func NewTCPRelay(
	src string,
	destination string,
	enableUDP bool,
	transport string,
	hostHeader string,
	wsPath string,
	serverType int,
) IService {
	listener, err := net.Listen("tcp", src)
	panicIfErr(err)

	var udpListener *net.UDPConn
	var connKeeper *ConnKeeper
	if enableUDP {
		uAddr, err := net.ResolveUDPAddr("udp", src)
		panicIfErr(err)
		uListener, err := net.ListenUDP("udp", uAddr)
		panicIfErr(err)
		udpListener = uListener
		connKeeper = &ConnKeeper{
			connections: make(map[string]net.Conn),
		}
	}

	return &TCPRelay{
		listener:      listener,
		src:           src,
		destination:   destination,
		udpListener:   udpListener,
		udpEnabled:    enableUDP,
		connKeeper:    connKeeper,
		udpBufferSize: 16 * 1024,
		transport:     transport,
		hostHeader:    hostHeader,
		wsPath:        wsPath,
		serverType:    serverType,
	}
}

func (s *TCPRelay) Start() {
	if s.udpEnabled && s.serverType == 2 {
		go func() {
			buf := make([]byte, s.udpBufferSize)
			for {
				nRead, cliAddr, err := s.udpListener.ReadFromUDP(buf)
				if err != nil {
					println(err.Error())
					s.connKeeper.Lock()
					if _, isThisConnInConnList := s.connKeeper.connections[cliAddr.String()]; isThisConnInConnList {
						delete(s.connKeeper.connections, cliAddr.String())
					}
					s.connKeeper.Unlock()
					continue
				}
				if nRead == 0 {
					continue
				}
				var connToGate net.Conn
				s.connKeeper.Lock()
				if c, isThere := s.connKeeper.connections[cliAddr.String()]; isThere {
					connToGate = c
					s.connKeeper.Unlock()
				} else {
					s.connKeeper.Unlock()
					println(fmt.Sprintf(`%s - [udp] %s <--> %s`, time.Now().Format(time.RFC3339), cliAddr.String(), s.destination))
					connToGate, err = net.DialTimeout("tcp", s.destination, 4*time.Second)
					if err != nil {
						println(err.Error())
						continue
					}
					if s.transport == "websocket" {
						err := s.handshakeWSFromRelay(connToGate)
						if err != nil {
							println(err.Error())
							return
						}
					}

					s.connKeeper.Lock()
					s.connKeeper.connections[cliAddr.String()] = connToGate
					s.connKeeper.Unlock()
					go s.fromGateToUDPClient(s.udpListener, connToGate, cliAddr)
					_, err = connToGate.Write([]byte("yyy"))
					if err != nil {
						println(err.Error())
						_ = connToGate.Close()
						continue
					}
				}

				_, err = connToGate.Write(buf[:nRead])
				if err != nil {
					_ = connToGate.Close()
				}
			}
		}()
	}
	for {
		srcConn, err := s.listener.Accept()
		if err != nil {
			println("accept error: ", err.Error())
			continue
		}

		//println(fmt.Sprintf(`%s - %s <--> %s`, time.Now().Format(time.RFC3339), srcConn.RemoteAddr().String(), s.destination))

		go func(srcConn net.Conn) {
			if s.serverType == 1 {
				if s.transport == "websocket" {
					err = s.handshakeWSFromGate(srcConn)
					if err != nil {
						println(err.Error())
						return
					}
				}
			}

			go func(srcConn net.Conn) {
				chosenNetwork := "tcp"
				headBuffer := make([]byte, 3)
				headNRead := 0
				if s.serverType == 1 && s.udpEnabled {
					var err error
					headNRead, err = srcConn.Read(headBuffer)
					if err != nil {
						println(err.Error())
						_ = srcConn.Close()
						return
					}
					if string(headBuffer) == "yyy" {
						chosenNetwork = "udp"
					}
				}

				println(fmt.Sprintf(`%s - [%s] %s <--> %s`, time.Now().Format(time.RFC3339), chosenNetwork, srcConn.RemoteAddr().String(), s.destination))

				destConn, err := net.DialTimeout(chosenNetwork, s.destination, 4*time.Second)

				if err != nil {
					println(err.Error())
					_ = srcConn.Close()
					return
				}

				if chosenNetwork == "tcp" {
					_, err := destConn.Write(headBuffer[:headNRead])
					if err != nil {
						println(err.Error())
						_ = destConn.Close()
						_ = srcConn.Close()
						return
					}
				}

				if s.serverType == 2 {
					if s.transport == "websocket" {
						err := s.handshakeWSFromRelay(destConn)
						if err != nil {
							_ = srcConn.Close()
							println(err.Error())
							return
						}
					}
				}
				go func(srcConn net.Conn, destConn net.Conn) {
					defer srcConn.Close()
					defer destConn.Close()
					_, err := io.Copy(destConn, srcConn)
					//println(n, "bytes from client")
					if err != nil {
						if strings.LastIndex(err.Error(), "closed network connection") >= 0 {
							return
						}
						println(err.Error())
					}
				}(srcConn, destConn)
				go func(srcConn net.Conn, destConn net.Conn) {
					defer srcConn.Close()
					defer destConn.Close()
					_, err := io.Copy(srcConn, destConn)
					//println(n, "bytes from server")
					if err != nil {
						if strings.LastIndex(err.Error(), "closed network connection") >= 0 {
							return
						}
						println(err.Error())
					}
				}(srcConn, destConn)
			}(srcConn)
		}(srcConn)
	}
}

func (s *TCPRelay) handshakeWSFromRelay(destConn net.Conn) error {
	if s.destination[strings.Index(s.destination, ":")+1:] == `443` {
		println("tls not supported")
		os.Exit(1)
	}
	if len(s.hostHeader) == 0 {
		s.hostHeader = s.destination[:strings.Index(s.destination, ":")]
		if s.destination[strings.Index(s.destination, ":")+1:] != `80` {
			s.hostHeader = s.destination
		}
	}
	socKey := base64Encode(RandString(16))
	wsReq := fmt.Sprintf(`GET %s HTTP/1.1
Host: %s
Connection: upgrade
Upgrade: websocket
Sec-WebSocket-Key: %s
Sec-WebSocket-Version: 13
Origin: http://%s
User-Agent: Go-http-client/1.1
Cache-Control: no-cache`, s.wsPath, s.hostHeader, socKey, s.hostHeader) + "\r\n\r\n"
	n, err := destConn.Write([]byte(wsReq))
	if err != nil {
		println(err.Error())
		_ = destConn.Close()
		return err
	}
	if n < len(wsReq) {
		println(len(wsReq)-n, "bytes was not wrote")
		_ = destConn.Close()
		return err
	}
	buf := make([]byte, 1024)
	n, err = destConn.Read(buf)
	if err != nil {
		println(err.Error())
		_ = destConn.Close()
		return err
	}
	if n == 0 {
		_ = destConn.Close()
		return errors.New("zero bytes were read")
	}
	bufStr := string(buf)
	if strings.LastIndex(bufStr, "\r\n\r\n") < 10 || strings.Index(bufStr, "101") < 4 {
		_ = destConn.Close()
		return errors.New("bad handshake, couldn't create tunnel through websocket")
	}
	return nil
}

func (s *TCPRelay) handshakeWSFromGate(srcConn net.Conn) error {
	buf := make([]byte, 1024)
	n, err := srcConn.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		_ = srcConn.Close()
		return errors.New("zero bytes received")
	}
	bufStr := string(buf)
	if strings.Index(bufStr, fmt.Sprintf(`GET %s `, s.wsPath)) < 0 || strings.Index(bufStr, "\r\n\r\n") < 0 {
		_, _ = srcConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		_ = srcConn.Close()
		return errors.New("invalid request")
	}
	socKey := base64Encode(RandString(16))
	wsResp := fmt.Sprintf(`HTTP/1.1 101 upgrade
Connection: upgrade
Upgrade: websocket
Sec-WebSocket-Accept: %s`, socKey) + "\r\n\r\n"
	n, err = srcConn.Write([]byte(wsResp))
	if err != nil {
		println(err.Error())
		_ = srcConn.Close()
		return err
	}
	if n != len(wsResp) {
		_ = srcConn.Close()
		return errors.New(fmt.Sprintf(`%d bytes was not wrote`, len(wsResp)-n))
	}
	return nil
}

func (s *TCPRelay) fromGateToUDPClient(conn *net.UDPConn, connToGate net.Conn, cliAddr *net.UDPAddr) {
	buf := make([]byte, s.udpBufferSize)
	for {
		connToGate.SetReadDeadline(time.Now().Add(8 * time.Second))
		n, err := connToGate.Read(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "timeout") {
				println(err.Error())
			}
			break
		}
		if n == 0 {
			break
		}
		nw, err := conn.WriteToUDP(buf[0:n], cliAddr)
		if err != nil {
			println(err.Error())
			break
		}
		if nw != n {
			println("wasn't able to write all the buffer")
			break
		}
	}
	s.connKeeper.Lock()
	_ = connToGate.Close()
	delete(s.connKeeper.connections, cliAddr.String())
	s.connKeeper.Unlock()
}
