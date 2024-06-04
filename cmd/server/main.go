package main

import (
	"crypto/tls"
	"log"

	"github.com/jenyaftw/trust/internal/app"
	"github.com/jenyaftw/trust/internal/pkg/crypto"
	"github.com/jenyaftw/trust/internal/pkg/flags"
)

var Clients = make(map[string]*tls.Conn)
var Peers = make(map[string]*tls.Conn)

func main() {
	flags := flags.ParseServerFlags()

	config, err := crypto.GetTLSConfig(flags.Cert, flags.Key, &flags.Ca)
	if err != nil {
		log.Println(err)
		return
	}

	app.ListenServer(flags, config)
}
