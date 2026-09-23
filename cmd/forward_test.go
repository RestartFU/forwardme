package main

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestTCPForwarding(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go func() {
		conn, err := backend.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	f, err := startForwarders(map[string]string{"127.0.0.1:0": backend.Addr().String()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	client, err := net.Dial("tcp", f.listeners[0].(net.Listener).Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	client.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 5)
	if _, err := io.ReadFull(client, got); err != nil || string(got) != "hello" {
		t.Fatalf("TCP reply = %q, %v", got, err)
	}
}

func TestUDPForwardingKeepsClientsSeparate(t *testing.T) {
	backend, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go func() {
		buf := make([]byte, 65535)
		for {
			n, addr, err := backend.ReadFromUDP(buf)
			if err != nil {
				return
			}
			backend.WriteToUDP(buf[:n], addr)
		}
	}()

	f, err := startForwarders(nil, map[string]string{"127.0.0.1:0": backend.LocalAddr().String()})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	addr := f.listeners[0].(*net.UDPConn).LocalAddr().(*net.UDPAddr)
	for _, message := range []string{"first", "second"} {
		client, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			t.Fatal(err)
		}
		client.SetDeadline(time.Now().Add(3 * time.Second))
		if _, err := client.Write([]byte(message)); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 32)
		n, err := client.Read(buf)
		client.Close()
		if err != nil || string(buf[:n]) != message {
			t.Fatalf("UDP reply = %q, %v", buf[:n], err)
		}
	}
}

func TestForwardersRejectInvalidTarget(t *testing.T) {
	if _, err := startForwarders(map[string]string{"127.0.0.1:0": "missing-port"}, nil); err == nil {
		t.Fatal("expected invalid target to fail startup")
	}
}
