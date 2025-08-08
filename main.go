package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"

	"github.com/InWILL/MioSocks/engine"
	"github.com/InWILL/MioSocks/proxy"
)

type Rules struct {
	Domain  []string `json:"domain,omitempty"`
	Process []string `json:"process,omitempty"`
}

type Config struct {
	Proxy     map[string]any `json:"proxy"`
	Rules     Rules          `json:"rules,omitempty"`
	ProxyFile string         `json:"proxyfile,omitempty"`
	RuleFile  string         `json:"rulefile,omitempty"`
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
	port := flag.String("p", "2801", "Port to listen the connection")
	config := flag.String("c", "config.json", "Path to the configuration file")

	flag.Parse()
	globalConfig := ParseConfig(*config)

	addr := ":" + *port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	defer ln.Close()

	dialer := proxy.NewProxy(globalConfig.Proxy)
	defer dialer.Close()

	log.Printf("%s server: %s listening on %s", dialer.Type(), dialer.Name(), addr)
	engine := engine.NewEngine(dialer)
	engine.Start()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go dialer.HandleConnection(conn)
	}
}
