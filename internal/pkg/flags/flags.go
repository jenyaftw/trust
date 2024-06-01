package flags

import (
	"flag"
	"log"
)

type ServerFlags struct {
	Host   string
	Port   string
	Cert   string
	Key    string
	CaFile string
	Peers  string
}

type ClientFlags struct {
	ServerHost string
	ServerPort string
	Cert       string
	Key        string
}

const (
	HOST  = "127.0.0.1"
	PORT  = "8736"
	PEERS = ""
)

func ParseServerFlags() *ServerFlags {
	host := flag.String("host", HOST, "Listening host")
	port := flag.String("port", PORT, "Listening port")

	cert := flag.String("cert", "certs/server-1.crt", "Certificate")
	key := flag.String("key", "certs/server-1.key", "Key")

	caFile := flag.String("cacert", "certs/ca.crt", "CA certificate")

	peers := flag.String("peers", PEERS, "Peers")

	if *cert == "" || *key == "" {
		log.Fatal("Certificate and key are required")
	}

	flag.Parse()

	return &ServerFlags{
		Host:   *host,
		Port:   *port,
		Cert:   *cert,
		Key:    *key,
		CaFile: *caFile,
		Peers:  *peers,
	}
}

func ParseClientFlags() *ClientFlags {
	host := flag.String("host", HOST, "Server host")
	port := flag.String("port", PORT, "Server port")

	cert := flag.String("cert", "certs/client-1.crt", "Certificate")
	key := flag.String("key", "certs/client-1.key", "Key")

	if *cert == "" || *key == "" {
		log.Fatal("Certificate and key are required")
	}

	flag.Parse()

	return &ClientFlags{
		ServerHost: *host,
		ServerPort: *port,
		Cert:       *cert,
		Key:        *key,
	}
}
