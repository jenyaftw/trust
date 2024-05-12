package app

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/message"
	"github.com/jenyaftw/trust/internal/pkg/utils"
)

var serverId uint64
var clients = make(map[uint64]*tls.Conn)
var peers = make(map[uint64]*tls.Conn)

func ListenServer(flags *flags.Flags, config *tls.Config) {
	serverId = utils.GenerateRandomId()
	fmt.Println("Server ID:", serverId)

	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%s", flags.Host, flags.Port), config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	peers := strings.Split(flags.Peers, ",")
	for _, peer := range peers {
		if peer != "" {
			go joinPeer(peer, config)
		}
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleServerConnection(conn.(*tls.Conn))
	}
}

func joinPeer(peer string, config *tls.Config) {
	conn, err := tls.Dial("tcp", peer, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Inform the client about the server ID
	msg := &message.Message{
		Type: message.PEER_ID,
		From: serverId,
	}
	msg.Send(conn)

	var peerId uint64
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(n, err)
			return
		}

		msg, err := message.MessageFromBytes(buf[:n])
		if err != nil {
			log.Println(err)
			return
		}

		switch msg.Type {
		case message.PEER_ID:
			fmt.Println(fmt.Sprintf("Peer ID (%s):", peer), msg.From)
			peerId = msg.From
			peers[peerId] = conn
		case message.PING:
			fmt.Println("Received ping from", peerId)
			msg := &message.Message{
				Type: message.PONG,
				From: serverId,
			}
			msg.Send(conn)
		case message.PONG:
			fmt.Println("Received pong from", peerId)
		}
	}
}

func handleServerConnection(conn *tls.Conn) {
	defer conn.Close()

	msg := &message.Message{
		Type: message.PEER_ID,
		From: serverId,
	}
	msg.Send(conn)

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(n, err)
			return
		}

		msg, err := message.MessageFromBytes(buf[:n])
		if err != nil {
			log.Println(err)
			return
		}

		switch msg.Type {
		case message.PEER_ID:
			fmt.Println("Peer ID:", msg.From)
			peers[msg.From] = conn
		case message.PING:
			fmt.Println("Received ping")
			msg := &message.Message{
				Type: message.PONG,
				From: serverId,
			}
			msg.Send(conn)
		case message.PONG:
			fmt.Println("Received pong")
		}
	}
}
