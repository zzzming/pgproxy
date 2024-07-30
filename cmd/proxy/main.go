package main

import (
	"log"
	"net"

	"github.com/zzzming/pgproxy/pkg/config"
	"github.com/zzzming/pgproxy/pkg/proxy"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}
	// Define the proxy listener address
	proxyAddr := ":5432"

	// Listen on the specified address
	listener, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", proxyAddr, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s", proxyAddr)

	// Handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go proxy.HandleConnection(conn, cfg)
	}
}
