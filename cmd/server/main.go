package main

import (
	"crypto/tls"
	"log"

	"github.com/jenyaftw/trust/internal/app"
	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/utils"
)

var Clients = make(map[string]*tls.Conn)
var Peers = make(map[string]*tls.Conn)

func main() {
	flags := flags.ParseFlags()

	config, err := utils.GetTLSConfig(flags.Cert, flags.Key, flags.CaFile)
	if err != nil {
		log.Println(err)
		return
	}

	app.ListenServer(flags, config)
}