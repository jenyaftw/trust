package flags

import (
	"flag"
	"log"
)

type ServerFlags struct {
	NodeId     int
	NodeCount  int
	Host       string
	Port       string
	Cert       string
	Key        string
	Ca         string
	Timeout    int
	Peers      string
	BufferSize int
}

type ClientFlags struct {
	ServerHost string
	ServerPort string
	Cert       string
	Key        string
	BufferSize int
}

const (
	HOST  = "127.0.0.1"
	PORT  = "8736"
	PEERS = ""
)

func ParseServerFlags() *ServerFlags {
	nodeId := flag.Int("id", 0, "Node ID")
	host := flag.String("host", HOST, "Listening host")
	port := flag.String("port", PORT, "Listening port")

	cert := flag.String("cert", "", "Certificate in Base64")
	key := flag.String("key", "", "Key in Base64")
	ca := flag.String("ca", "", "CA certificate in Base64")
	timeout := flag.Int("timeout", 5000, "Timeout for connection")
	bufferSize := flag.Int("buffer", 64*1024, "Buffer size")

	peers := flag.String("peers", PEERS, "Peers")
	nodes := flag.Int("nodes", 0, "Number of nodes")

	flag.Parse()

	return &ServerFlags{
		Host:       *host,
		Port:       *port,
		Cert:       *cert,
		Key:        *key,
		Ca:         *ca,
		Peers:      *peers,
		NodeId:     *nodeId,
		NodeCount:  *nodes,
		Timeout:    *timeout,
		BufferSize: *bufferSize,
	}
}

func ParseClientFlags() *ClientFlags {
	host := flag.String("host", HOST, "Server host")
	port := flag.String("port", PORT, "Server port")
	bufferSize := flag.Int("buffer", 64*1024, "Buffer size")

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
		BufferSize: *bufferSize,
	}
}
