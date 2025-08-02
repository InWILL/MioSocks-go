package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
)

type Config struct {
	Proxy   map[string]any `json:"proxy"`
	Domain  []string       `json:"domain,omitempty"`
	Process []string       `json:"process,omitempty"`
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
	// a := &Config{
	// 	Proxy: map[string]any{
	// 		"name":             "vmess-node",
	// 		"type":             "vmess",
	// 		"server":           "127.0.0.1",
	// 		"port":             2801,
	// 		"uuid":             "3e976926-ea42-4dc8-99f5-560803dc573c",
	// 		"alterId":          0,
	// 		"cipher":           "auto",
	// 		"tls":              false,
	// 		"skip-cert-verify": true,
	// 		"udp":              true,
	// 	},
	// 	Domain: []string{
	// 		"DOMAIN-SUFFIX,baidu.com",
	// 		"DOMAIN-SUFFIX,google.com",
	// 	},
	// 	Process: []string{
	// 		"MapleStory.exe",
	// 	},
	// }
	// data, err := json.MarshalIndent(a, "", "  ")
	// if err != nil {
	// 	log.Fatalf("Failed to marshal default config: %v", err)
	// }
	// os.WriteFile("config.json", data, 0644)

	// return

	port := flag.String("p", "2801", "Port to run the server on")
	config := flag.String("c", "config.json", "Path to the configuration file")

	flag.Parse()
	globalConfig := ParseConfig(*config)

	addr := ":" + *port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go globalConfig.handleConnection(conn)
	}
}
