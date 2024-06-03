package app

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/message"
	"github.com/jenyaftw/trust/internal/pkg/utils"
)

var serverId int
var clients = make(map[uint64]*tls.Conn)
var peers = make(map[uint64]*tls.Conn)

func ListenServer(flags *flags.ServerFlags, config *tls.Config) {
	serverId = flags.NodeId
	fmt.Println("Current server ID:", serverId)

	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%s", flags.Host, flags.Port), config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	time.Sleep(time.Duration(flags.Timeout * 1_000_000))

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
		go handleConnection(conn.(*tls.Conn))
	}
}

func joinPeer(peer string, config *tls.Config) {
	conn, err := tls.Dial("tcp", peer, config)
	if err != nil {
		log.Fatal(err)
		return
	}

	go handleConnection(conn)
}

func handleConnection(conn *tls.Conn) {
	defer conn.Close()

	msg := &message.Message{
		Type: message.PEER_ID,
		From: uint64(serverId),
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
				From: uint64(serverId),
			}
			msg.Send(conn)
		case message.PONG:
			fmt.Println("Received pong")
		case message.REGISTER_CLIENT:
			fmt.Println("Registering new client")
			clientId := utils.GenerateRandomId()
			clients[clientId] = conn
			msg := &message.Message{
				Type: message.REGISTER_CLIENT_RESP,
				From: uint64(serverId),
				To:   clientId,
			}
			msg.Send(conn)
		case message.GET_CLIENT_CERT:
			fmt.Println("Received request for client certificate")
			clientConn, ok := clients[msg.To]
			if !ok {
				fmt.Println("We don't have the client, let's ask peers")
				for _, peer := range peers {
					peer.Write(buf[:n])
				}
				continue
			}
			clientConn.Write(buf[:n])
		case message.DATA:
			fmt.Println("Received message data")
			clientConn, ok := clients[msg.To]
			if !ok {
				fmt.Println("We don't have the client, let's ask peers")
				for _, peer := range peers {
					peer.Write(buf[:n])
				}
				continue
			}
			clientConn.Write(buf[:n])
		case message.GET_CLIENT_CERT_RESP:
			fmt.Println("Received request for client certificate response")
			clientConn, ok := clients[msg.To]
			if !ok {
				fmt.Println("We don't have the client, let's ask peers")
				for _, peer := range peers {
					peer.Write(buf[:n])
				}
				continue
			}
			clientConn.Write(buf[:n])
		}
	}
}
