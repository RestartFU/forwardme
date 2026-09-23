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

func startForwarders(tcpRoutes map[string]string) (*forwarders, error) {
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
