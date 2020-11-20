package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	usageFmt = "%s dest msg\n"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, usageFmt, os.Args[0])
		return 1
	}
	dest := os.Args[1]
	msg := os.Args[2]

	r, err := icmpEcho(dest, msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("received:", string(r))

	return 0
}

func icmpEcho(addr, msg string) (r []byte, err error) {
	// Start listening for icmp replies
	var c *icmp.PacketConn
	c, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return
	}
	defer func() {
		e := c.Close()
		if err != nil {
			err = e
		}
	}()

	// Resolve any DNS (if used) and get the real IP of the target
	var dst *net.IPAddr
	dst, err = net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return
	}

	// Make a new ICMP message
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1, //<< uint(seq), // TODO
			Data: []byte(msg),
		},
	}
	var bb []byte
	bb, err = m.Marshal(nil)
	if err != nil {
		return
	}

	//fmt.Println(hex.Dump(bb))

	var n int
	n, err = c.WriteTo(bb, dst)
	//fmt.Println("n =", n)
	if err != nil {
		return
	} else if n != len(bb) {
		err = fmt.Errorf("got %v; want %v", n, len(bb))
		return
	}

	// Wait for a reply
	reply := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return
	}

	var peer net.Addr
	var n2 int
	n2, peer, err = c.ReadFrom(reply)
	if err != nil {
		return
	}

	//fmt.Println("peer =", peer.String())
	//fmt.Println("n2 =", n2)
	//fmt.Println(hex.Dump(reply[:n2]))

	// Pack it up boys, we're done here
	var rm *icmp.Message
	rm, err = icmp.ParseMessage(1 /*ICMP*/, reply[:n2])
	if err != nil {
		return
	}

	if rm.Type != ipv4.ICMPTypeEchoReply {
		err = fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
		return
	}

	switch rm.Body.(type) {
	case *icmp.Echo:
		body, _ := rm.Body.(*icmp.Echo)
		//fmt.Println(string(body.Data))
		r = body.Data
	default:
		err = fmt.Errorf("not Echo message")
	}

	/*
			var body *icmp.Echo
			var ok bool
			body, ok = rm.Body.(*icmp.Echo)
		fmt.Println(ok, body.Data)
	*/

	//bb, err = rm.Body.Marshal(1 /*ICMP*/)
	//fmt.Println(bb)
	//fmt.Println(rm.Body.Len(1))
	return
}
