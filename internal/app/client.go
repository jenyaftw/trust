package app

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/message"
	"github.com/jenyaftw/trust/internal/pkg/utils"
)

type TrustClient struct {
	flags    *flags.ClientFlags
	config   *tls.Config
	conn     *tls.Conn
	clientId uint64
	serverId uint64
	certs    map[uint64]*x509.Certificate
}

func NewTrustClient(flags *flags.ClientFlags) (*TrustClient, error) {
	config, err := utils.GetTLSConfig(flags.Cert, flags.Key, nil)
	if err != nil {
		return nil, err
	}

	return &TrustClient{config: config, flags: flags, certs: make(map[uint64]*x509.Certificate)}, nil
}

func (c *TrustClient) Connect() error {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", c.flags.ServerHost, c.flags.ServerPort), c.config)
	if err != nil {
		return err
	}

	c.conn = conn

	v := make(chan uint64)
	go c.handleConnection(v)
	<-v

	return nil
}

func (c *TrustClient) Send(bytes []byte, dest uint64) error {
	cert := c.certs[dest]
	if cert == nil {
		msg := &message.Message{
			Type:    message.GET_CLIENT_CERT,
			Content: c.config.Certificates[0].Certificate[0],
			From:    c.clientId,
			To:      dest,
		}

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
	}

	encrypted, err := utils.EncryptMessage(bytes, cert)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:    message.DATA,
		Content: encrypted,
		From:    c.clientId,
		To:      dest,
	}

	return msg.Send(c.conn)
}

func (c *TrustClient) Close() {
	c.conn.Close()
}

func (c *TrustClient) handleConnection(v chan uint64) {
	defer c.Close()

	for {
		buf := make([]byte, 1024)
		n, err := c.conn.Read(buf)
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
				Type:    message.GET_CLIENT_CERT_RESP,
				Content: c.config.Certificates[0].Certificate[0],
				From:    c.clientId,
				To:      msg.From,
			}
			msg.Send(c.conn)
		case message.GET_CLIENT_CERT_RESP:
			cert, err := x509.ParseCertificate(msg.Content)
			if err != nil {
				fmt.Println(err)
				continue
			}
			c.certs[msg.From] = cert
		case message.DATA:
			key := c.config.Certificates[0].PrivateKey.(*rsa.PrivateKey)
			key.Precompute()
			decrypted, err := utils.DecryptMessage(msg.Content, key)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("Received message from", msg.From, ":", string(decrypted))
		}
	}
}
