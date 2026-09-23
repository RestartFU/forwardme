package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const udpIdleTimeout = time.Minute

type forwarders struct {
	listeners []io.Closer
}

func (f *forwarders) Close() error {
	var errs []error
	for _, listener := range f.listeners {
		errs = append(errs, listener.Close())
	}
	return errors.Join(errs...)
}

func startForwarders(tcpRoutes, udpRoutes map[string]string) (*forwarders, error) {
	f := &forwarders{}
	for listen, target := range tcpRoutes {
		if err := validateTarget(target); err != nil {
			f.Close()
			return nil, fmt.Errorf("TCP route %q: %w", listen, err)
		}
		listener, err := net.Listen("tcp", listen)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("TCP listen %q: %w", listen, err)
		}
		f.listeners = append(f.listeners, listener)
		go serveTCP(listener, target)
		log.Printf("TCP %s -> %s", listener.Addr(), target)
	}
	for listen, target := range udpRoutes {
		if err := validateTarget(target); err != nil {
			f.Close()
			return nil, fmt.Errorf("UDP route %q: %w", listen, err)
		}
		addr, err := net.ResolveUDPAddr("udp", listen)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("UDP listen %q: %w", listen, err)
		}
		listener, err := net.ListenUDP("udp", addr)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("UDP listen %q: %w", listen, err)
		}
		f.listeners = append(f.listeners, listener)
		go serveUDP(listener, target)
		log.Printf("UDP %s -> %s", listener.LocalAddr(), target)
	}
	return f, nil
}

func validateTarget(target string) error {
	host, port, err := net.SplitHostPort(target)
	if err != nil || host == "" || port == "" {
		return fmt.Errorf("target %q must be host:port", target)
	}
	return nil
}

func serveTCP(listener net.Listener, target string) {
	for {
		client, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("TCP accept on %s: %v", listener.Addr(), err)
			}
			return
		}
		go func() {
			backend, err := net.DialTimeout("tcp", target, 10*time.Second)
			if err != nil {
				log.Printf("TCP dial %s: %v", target, err)
				client.Close()
				return
			}
			defer client.Close()
			defer backend.Close()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				io.Copy(backend, client)
				backend.(*net.TCPConn).CloseWrite()
			}()
			io.Copy(client, backend)
			client.(*net.TCPConn).CloseWrite()
			wg.Wait()
		}()
	}
}

type udpSession struct {
	backend *net.UDPConn
	client  *net.UDPAddr
}

func serveUDP(listener *net.UDPConn, target string) {
	sessions := make(map[string]*udpSession)
	defer func() {
		for _, session := range sessions {
			session.backend.Close()
		}
	}()
	// Only this loop changes sessions. Replies report expired sessions through done.
	done := make(chan *udpSession, 1024)
	buf := make([]byte, 65535)
	for {
		_ = listener.SetReadDeadline(time.Now().Add(time.Second))
		n, client, err := listener.ReadFromUDP(buf)
		for {
			select {
			case session := <-done:
				key := session.client.String()
				if sessions[key] == session {
					delete(sessions, key)
				}
			default:
				goto drained
			}
		}
	drained:
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				continue
			}
			log.Printf("UDP read on %s: %v", listener.LocalAddr(), err)
			continue
		}
		key := client.String()
		session := sessions[key]
		if session == nil {
			backend, err := net.DialTimeout("udp", target, 10*time.Second)
			if err != nil {
				log.Printf("UDP dial %s: %v", target, err)
				continue
			}
			session = &udpSession{backend: backend.(*net.UDPConn), client: client}
			sessions[key] = session
			go relayUDPReplies(listener, session, done)
		}
		_ = session.backend.SetReadDeadline(time.Now().Add(udpIdleTimeout))
		if _, err := session.backend.Write(buf[:n]); err != nil {
			log.Printf("UDP write to %s: %v", target, err)
			session.backend.Close()
			delete(sessions, key)
		}
	}
}

func relayUDPReplies(listener *net.UDPConn, session *udpSession, done chan<- *udpSession) {
	defer session.backend.Close()
	defer func() { done <- session }()
	buf := make([]byte, 65535)
	for {
		n, err := session.backend.Read(buf)
		if err != nil {
			return
		}
		if _, err := listener.WriteToUDP(buf[:n], session.client); err != nil {
			return
		}
	}
}
