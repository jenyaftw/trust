package message

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
)

type Message struct {
	Type     uint8
	From, To uint64
	Content  []byte
}

const (
	PEER_ID              uint8 = 0
	REGISTER_CLIENT      uint8 = 1
	REGISTER_CLIENT_RESP uint8 = 2
	DATA                 uint8 = 3
	CLIENT_NON_EXISTENT  uint8 = 4
	PING                 uint8 = 5
	PONG                 uint8 = 6
	GET_CLIENT_CERT      uint8 = 7
	GET_CLIENT_CERT_RESP uint8 = 8
)

func MessageFromBytes(input []byte) (*Message, error) {
	dec := gob.NewDecoder(bytes.NewReader(input))
	msg := &Message{}
	err := dec.Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (m *Message) Send(conn *tls.Conn) error {
	msgBytes, err := m.Bytes()
	if err != nil {
		return err
	}
	_, err = conn.Write(msgBytes)
	if err != nil {
		return err
	}
	return nil
}

func (m *Message) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
