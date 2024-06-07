package app

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/crypto"
	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/message"
	"github.com/jenyaftw/trust/internal/pkg/structs"
)

type TrustClient struct {
	flags              *flags.ClientFlags
	config             *tls.Config
	conn               *tls.Conn
	clientId           uint64
	serverId           uint64
	certs              map[uint64]*x509.Certificate
	keys               map[uint64][]byte
	channels           []chan []byte
	blockchains        map[uint64]*structs.Blockchain
	validateBlockchain bool
}

func NewTrustClient(flags *flags.ClientFlags) (*TrustClient, error) {
	certContent, err := os.ReadFile(flags.Cert)
	if err != nil {
		fmt.Print(err)
	}

	keyContent, err := os.ReadFile(flags.Key)
	if err != nil {
		fmt.Print(err)
	}

	certEnc := base64.StdEncoding.EncodeToString(certContent)
	keyEnc := base64.StdEncoding.EncodeToString(keyContent)

	config, err := crypto.GetTLSConfig(certEnc, keyEnc, nil)
	if err != nil {
		return nil, err
	}

	return &TrustClient{config: config, flags: flags, validateBlockchain: flags.ValidateBlockchain, certs: make(map[uint64]*x509.Certificate), keys: make(map[uint64][]byte), blockchains: make(map[uint64]*structs.Blockchain)}, nil
}

func (c *TrustClient) Connect(bufferSize int) error {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", c.flags.ServerHost, c.flags.ServerPort), c.config)
	if err != nil {
		return err
	}

	c.conn = conn

	v := make(chan uint64)
	go c.handleConnection(v, bufferSize)
	<-v

	return nil
}

func (c *TrustClient) Read() chan []byte {
	ch := make(chan []byte)
	c.channels = append(c.channels, ch)
	return ch
}

func (c *TrustClient) Send(bytes []byte, dest uint64) error {
	cert := c.certs[dest]
	key := c.keys[dest]
	blockchain := c.blockchains[dest]

	if blockchain == nil {
		blockchain = structs.NewBlockchain()
		c.blockchains[dest] = blockchain
	}

	if cert == nil {
		msg := &message.Message{
			Type:         message.GET_CLIENT_CERT,
			Content:      c.config.Certificates[0].Certificate[0],
			From:         c.clientId,
			To:           dest,
			Intermediate: -1,
		}

		fmt.Println("Requesting client cert from", dest)
		err := msg.Send(c.conn)
		if err != nil {
			return err
		}

		i := 0
		for {
			if c.certs[dest] != nil {
				break
			}
			if i > 30 {
				return fmt.Errorf("request for client cert timed out")
			}
			i++
			time.Sleep(1000 * time.Millisecond)
		}
		cert = c.certs[dest]

		key = crypto.GenerateAESKey()
		c.keys[dest] = key
		aesKeyEncrypted, err := crypto.EncryptMessage(key, cert)
		if err != nil {
			return err
		}

		msg = &message.Message{
			Type:         message.AES_KEY,
			Content:      aesKeyEncrypted,
			From:         c.clientId,
			To:           dest,
			Intermediate: -1,
		}

		fmt.Println("Sending AES key to", dest)
		err = msg.Send(c.conn)
		if err != nil {
			return err
		}
	}

	bytesToSend := bytes

	if c.validateBlockchain {
		block := blockchain.AddBlockFromBytes(bytes)
		tree := structs.BuildTreeFromBlockchain(blockchain)
		block.MerkleRoot = tree.Root.Value
		fmt.Println("Merkle root:", block.MerkleRoot)

		bytes, err := block.Bytes()
		if err != nil {
			return err
		}
		bytesToSend = bytes
	}

	encrypted, err := crypto.EncryptMessageAES(bytesToSend, key)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:         message.DATA,
		Content:      encrypted,
		From:         c.clientId,
		To:           dest,
		Intermediate: -1,
	}

	return msg.Send(c.conn)
}

func (c *TrustClient) Close() {
	c.conn.Close()
}

func (c *TrustClient) handleConnection(v chan uint64, bufferSize int) {
	defer c.Close()

	for {
		size := 0
		n := 0
		buf := make([]byte, 0, bufferSize)
		for {
			if size != 0 && n >= size {
				break
			}

			newBuf := make([]byte, bufferSize)
			bytesRead, err := c.conn.Read(newBuf)
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

		switch msg.Type {
		case message.PEER_ID:
			fmt.Println("Received peer ID:", msg.From)
			c.serverId = msg.From
			msg := &message.Message{
				Type: message.REGISTER_CLIENT,
			}
			msg.Send(c.conn)
		case message.REGISTER_CLIENT_RESP:
			fmt.Println("Received client ID:", msg.To)
			c.clientId = msg.To
			v <- c.clientId
		case message.GET_CLIENT_CERT:
			cert, err := x509.ParseCertificate(msg.Content)
			if err != nil {
				fmt.Println(err)
				continue
			}
			c.certs[msg.From] = cert

			msg := &message.Message{
				Type:         message.GET_CLIENT_CERT_RESP,
				Content:      c.config.Certificates[0].Certificate[0],
				From:         c.clientId,
				To:           msg.From,
				Intermediate: -1,
			}
			msg.Send(c.conn)
		case message.GET_CLIENT_CERT_RESP:
			cert, err := x509.ParseCertificate(msg.Content)
			if err != nil {
				fmt.Println(err)
				continue
			}

			c.certs[msg.From] = cert
		case message.AES_KEY:
			key := c.config.Certificates[0].PrivateKey.(*rsa.PrivateKey)
			key.Precompute()
			aesKey, err := crypto.DecryptMessage(msg.Content, key)
			if err != nil {
				fmt.Println(err)
				continue
			}
			c.keys[msg.From] = aesKey
			c.blockchains[msg.From] = structs.NewBlockchain()
		case message.DATA:
			decrypted, err := crypto.DecryptMessageAES(msg.Content, c.keys[msg.From])
			if err != nil {
				fmt.Println(err)
				continue
			}

			if c.validateBlockchain {
				block, err := structs.DecodeBlock(decrypted)
				if err != nil {
					fmt.Println(err)
					continue
				}

				blockchain := c.blockchains[msg.From]
				blockchain.AddBlock(block)

				tree := structs.BuildTreeFromBlockchain(blockchain)
				if !bytes.Equal(tree.Root.Value, block.MerkleRoot) {
					fmt.Println("Merkle root mismatch")
					continue
				}
			}

			for _, ch := range c.channels {
				ch <- decrypted
			}
		}
	}
}
