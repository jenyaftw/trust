package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
)

const (
	HOST = "127.0.0.1"
	PORT = "8736"
)

func main() {
	host := flag.String("host", HOST, "Server host")
	port := flag.String("port", PORT, "Server port")

	cert := flag.String("cert", "certs/client-cert.pem", "Client certificate")
	key := flag.String("key", "certs/client-key.pem", "Client key")

	if *cert == "" || *key == "" {
		log.Fatal("Client certificate and key are required")
	}

	flag.Parse()

	cer, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", *host, *port), config)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
}
