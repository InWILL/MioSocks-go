package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
)

type Rules struct {
	Name    string   `json:"name"`
	Domain  []string `json:"domain,omitempty"`
	Process []string `json:"process,omitempty"`
}

type Config struct {
	Proxy map[string]any `json:"proxy"`
	Rules Rules          `json:"rules,omitempty"`
}

func ParseConfig(file string) *Config {
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		panic(err)
	}

	return config
}

func main() {
	port := flag.String("p", "2801", "Port to run the server on")
	config := flag.String("c", "config.json", "Path to the configuration file")

	flag.Parse()
	globalConfig := ParseConfig(*config)

	addr := ":" + *port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	defer ln.Close()

	proxy := NewProxy(globalConfig.Proxy)
	defer proxy.Close()

	log.Printf("%s server: %s listening on %s", proxy.Type(), proxy.Name(), addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go proxy.handleConnection(conn)
	}
}
