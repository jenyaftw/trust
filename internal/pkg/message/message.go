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
	PEER_ID             uint8 = 0
	CLIENT_ID           uint8 = 1
	DATA                uint8 = 2
	CLIENT_NON_EXISTENT uint8 = 3
	PING                uint8 = 4
	PONG                uint8 = 5
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
