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

	f, err := startForwarders(map[string]string{"127.0.0.1:0": backend.Addr().String()})
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

func TestForwardersRejectInvalidTarget(t *testing.T) {
	if _, err := startForwarders(map[string]string{"127.0.0.1:0": "missing-port"}); err == nil {
		t.Fatal("expected invalid target to fail startup")
	}
}
