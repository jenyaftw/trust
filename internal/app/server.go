package app

import (
	"crypto/tls"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/message"
	"github.com/jenyaftw/trust/internal/pkg/utils"
)

var serverId int
var clients = make(map[uint64]*tls.Conn)
var peers = make(map[uint64]*tls.Conn)
var clientNode = make(map[uint64]uint64)

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
			go joinPeer(peer, config, flags.NodeCount, flags.BufferSize)
		}
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn.(*tls.Conn), flags.NodeCount, flags.BufferSize)
	}
}

func joinPeer(peer string, config *tls.Config, nodeCount int, bufferSize int) {
	conn, err := tls.Dial("tcp", peer, config)
	if err != nil {
		log.Fatal(err)
		return
	}

	go handleConnection(conn, nodeCount, bufferSize)
}

func processMessageRelay(msg *message.Message, nodeCount int) uint64 {
	node := clientNode[msg.To]
	msg.FromNode = uint64(serverId)
	msg.ToNode = node

	bitCount := utils.GetBitCount(nodeCount - 1)
	allMask, _, _ := utils.GetMasks(bitCount)
	fmt.Println("All mask:", strconv.FormatInt(int64(allMask), 2))
	fmt.Println("From node:", strconv.FormatInt(int64(msg.FromNode), 2), "=", msg.FromNode)
	fmt.Println("To node:", strconv.FormatInt(int64(msg.ToNode), 2), "=", msg.ToNode)
	fmt.Println("Intermediate:", strconv.FormatInt(int64(msg.Intermediate), 2), "=", msg.Intermediate)
	if msg.Intermediate == -1 {
		msg.Intermediate = int64(msg.ToNode)
	}
	fmt.Println("Intermediate:", strconv.FormatInt(int64(msg.Intermediate), 2), "=", msg.Intermediate)

	shiftFrom := (uint64(serverId) << 1) & uint64(allMask)
	fmt.Println("Shift from:", strconv.FormatInt(int64(shiftFrom), 2), "=", shiftFrom)

	if msg.Intermediate != 0 {
		firstBit := utils.GetFirstBit(int(msg.Intermediate), nodeCount-1)
		fmt.Println("First bit:", strconv.FormatInt(int64(firstBit), 2), "=", firstBit)

		if firstBit == 1 {
			shiftFrom |= 1
		}
		fmt.Println("New shift from (next node):", strconv.FormatInt(int64(shiftFrom), 2), "=", shiftFrom)

		newIntermediate := (uint64(msg.Intermediate) << 1) & uint64(allMask)
		msg.Intermediate = int64(newIntermediate)
		fmt.Println("New intermediate:", strconv.FormatInt(int64(newIntermediate), 2), "=", newIntermediate)
		fmt.Println()
	}

	if shiftFrom == uint64(serverId) {
		return processMessageRelay(msg, nodeCount)
	}

	return shiftFrom
}

func handleConnection(conn *tls.Conn, nodeCount int, bufferSize int) {
	defer conn.Close()

	msg := &message.Message{
		Type: message.PEER_ID,
		From: uint64(serverId),
	}
	msg.Send(conn)

	for {
		size := 0
		n := 0
		buf := make([]byte, 0, bufferSize)
		for {
			if size != 0 && n >= size {
				break
			}

			newBuf := make([]byte, bufferSize)
			bytesRead, err := conn.Read(newBuf)
			if err != nil {
				log.Println(n, err)
				return
			}

			if size == 0 {
				size = int(message.ReadSize(newBuf[:4]))
			}

			buf = append(buf, newBuf...)
			n += bytesRead
		}

		msg, err := message.MessageFromBytes(buf[4:n])
		if err != nil {
			log.Println(err)
			continue
		}

		fmt.Println("Received message:", msg.Type, "from", msg.From, "to", msg.To)
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

			msg = &message.Message{
				Type:        message.I_HAVE_CLIENT,
				From:        uint64(serverId),
				To:          uint64(clientId),
				AlreadyBeen: []uint64{uint64(serverId)},
			}
			bytes, err := msg.Bytes()
			if err != nil {
				log.Println(err)
			}

			for id, peer := range peers {
				if !slices.Contains(msg.AlreadyBeen, id) {
					peer.Write(bytes)
				}
			}
		case message.I_HAVE_CLIENT:
			if _, ok := clientNode[msg.To]; ok {
				continue
			}

			fmt.Printf("I'm %d, I know that %d has client %d\n", serverId, msg.From, msg.To)
			clientNode[msg.To] = msg.From

			msg.AlreadyBeen = append(msg.AlreadyBeen, uint64(serverId))
			bytes, err := msg.Bytes()
			if err != nil {
				log.Println(err)
			}

			for id, peer := range peers {
				if !slices.Contains(msg.AlreadyBeen, id) {
					peer.Write(bytes)
				}
			}
			continue
		case message.GET_CLIENT_CERT:
			fmt.Println("Received request for client certificate")
			clientConn, ok := clients[msg.To]
			if !ok {
				nextNode := processMessageRelay(msg, nodeCount)

				peer := peers[nextNode]
				fmt.Println("Relaying to:", nextNode)
				if err := msg.Send(peer); err != nil {
					log.Println(err)
				}

				continue
			}
			clientConn.Write(buf[:n])
		case message.DATA:
			fmt.Println("Received message data")
			clientConn, ok := clients[msg.To]
			if !ok {
				nextNode := processMessageRelay(msg, nodeCount)

				peer := peers[nextNode]
				fmt.Println("Relaying to:", nextNode)
				if err := msg.Send(peer); err != nil {
					log.Println(err)
				}

				continue
			}
			clientConn.Write(buf[:n])
		case message.AES_KEY:
			fmt.Println("Received AES key")
			clientConn, ok := clients[msg.To]
			if !ok {
				nextNode := processMessageRelay(msg, nodeCount)

				peer := peers[nextNode]
				fmt.Println("Relaying to:", nextNode)
				if err := msg.Send(peer); err != nil {
					log.Println(err)
				}

				continue
			}
			clientConn.Write(buf[:n])
		case message.GET_CLIENT_CERT_RESP:
			fmt.Println("Received request for client certificate response")
			clientConn, ok := clients[msg.To]
			if !ok {
				nextNode := processMessageRelay(msg, nodeCount)

				peer := peers[nextNode]
				fmt.Println("Relaying to:", nextNode)
				if err := msg.Send(peer); err != nil {
					log.Println(err)
				}

				continue
			}
			clientConn.Write(buf[:n])
		}
	}
}
