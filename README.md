# tunnel

tunnel? well, its a tcp based relay.

It consists of two parts: `Relay` and `Gate`

---
### How It Works

as mentioned before it consists of two parts relay and gate.<br/>
you need to run the `tunnel` as `relay` in the server you want to accept connections from the `client` and run the `tunnel` as `gate` on the `exit node`( the host/vps you want to tunnel the connections to ) so the relay can establish a tunnel to gate to route the incoming connections to the destination.<br/>

#### Transport

the `relay` node is forwarding the connections to `gate` through the established tunnel, this tunnel can be established using `tcp` or `websocket` transport ( its tcp by default ) .<br/>
websocket path can be set using `-ws-path /path` when the transport is `websocket`

- some notes 
  - the transport type is basically tcp and thats why i said its a tcp based relay before
  - udp is getting transported over tcp then :D


#### UDP

udp tunneling is supported but the feature is experimental for now<br/>
udp can be enabled by passing the `-udp` in the params in both relay and gate sides.

if enabled, `relay` will accept udp and tcp on the same port and `gate` will forward it to the same destination.

### CDN
**Is Over CDN Tunneling Supported?**<br/>

yes over cdn tunneling is supported too, you need to create an A Record Pointing to the `gate` server with Proxy Enabled.</br>
just dont forget to set the hostname using `-host-header hostname`<br/>
**Can We Tunnel UDP Over CDN Too?**<br/>
yes Just Like when it's not behind cdn with no magic.<br/>
note: don't fotget to set the transport to `websocket` if you're going to use cdn between relay and gate

---


### Example

lets say, you want to create a tunnel from an ir vps to a xray service running on an ams vps.<br/>
so the `ir vps` is the `relay` because its going to accept the connections from the `client`.<br/>
and the ams ( amsterdam ) is the `gate` because its going to accept connections from `relay` and forward them to xray.<br/>

i assume :
  * xray is running on `127.0.0.1:4444` in the ams vps.
  * you want to accept the incoming client connections on `0.0.0.0:4433` in ir vps.
  * ams address is `ams.server.address.com` and you want to accept the relay requests in gate (ams) on `0.0.0.0:5555`

so after downloading the tunnel on both servers like this :
```
cd ~
wget https://github.com/benyamin218118/tunnel/raw/main/tunnel
chmod +x ./tunnel
ln -s /root/tunnel /bin/tunnel
```

**On IR VPS :**

```
$ tunnel -udp -type 2 -src 0.0.0.0:4433 -dst ams.server.address.com:5555
```
the `-udp` flags tells it to enable udp tunneling and `-type 2` is setting the server type to `relay` ( 2 for relay and 1 for gate )<br/>
here `-dst` is pointing to the ams server and the port is the gate port ( tunnel -src port as gate on ams ).

**On AMS VPS :**

```
$ tunnel -udp -type 1 -src 0.0.0.0:5555 -dst 127.0.0.1:4444
```
the `-udp` flags tells it to enable udp tunneling and `-type 1` is setting the server type to `gate` ( 2 for relay and 1 for gate )<br/>
here `-dst` is pointing to xray running on the ams server.

**On Client**

just set the xray config address to your IR VPS address and the port to 4433 ( because relay is accepting connections on 4433 ;D )


## How to Keep The Process Alive?
you can use screen or create a service for it
if service is the choice then you can create a service like this in **both servers**:

first you need to create a unit file in this address :<br/>
`/etc/systemd/system/SERVICENAME.service`

choose a service name and replace it with SERVICENAME first; lets use `tunnelsvc`<br/>
now you need to create the file with nano :<br/>
`nano /etc/systemd/system/tunnelsvc.service`

and paste this content into it :
```
[Unit]
Description=tunnel service
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=16
User=root
ExecStart=tunnel  ARGUMENTS

[Install]
WantedBy=multi-user.target
```
dont forget to edit the `ExecStart` value, thats the tunnel command you want to run as relay and gate.
after saving the contents ( by ctrl+x  y  enter ) you need to enable this `tunnelsvc` using `systemctl` so it will start again after reboot<br/>
`$ systemctl enable tunnelsvc`

and then you need to start the service<br/>
`$ service tunnelsvc start`

to check the service state you can use the `service tunnelsvc status` but if you wanned to see request logs :<br/>
`$ journalctl -u tunnelsvc -n 32 -f`


### Some Use cases

- tunneling ssh over cdn
- tunneling wireguard/openvpn/shadowsocks/xray protocols
- tunneling any tcp/udp based connection
---

# FAQ

- why does it log too many open files sometimes? how to fix it?
```
The "Too Many Open Files" error indicates that this process has reached its max open socket limit.
you can check the current open file limit (open socket limit in this case) using  `ulimit -a | grep open`

to fix this issue you need to change this limit to a higher number before running the tunnel
for example to set the limit to 10240 :
ulimit -n 10240
```
