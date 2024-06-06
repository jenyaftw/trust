package message

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"fmt"
)

type Message struct {
	Type             uint8
	From, To         uint64
	Intermediate     int64
	FromNode, ToNode uint64
	Content          []byte
	AlreadyBeen      []uint64
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
	I_HAVE_CLIENT        uint8 = 9
	AES_KEY              uint8 = 10
)

func MessageFromBytes(input []byte) (*Message, error) {
	dec := gob.NewDecoder(bytes.NewReader(input))
	msg := &Message{}
	err := dec.Decode(msg)
	if err != nil {
		return nil, fmt.Errorf("error decoding message: %v", err)
	}
	return msg, nil
}

func ReadSize(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes[:4])
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
		return nil, fmt.Errorf("error encoding message: %v", err)
	}
	bytes := buf.Bytes()

	headerSize := 4
	msgSize := len(bytes)
	header := make([]byte, headerSize)
	binary.BigEndian.PutUint32(header, uint32(msgSize))

	return append(header, bytes...), nil
}
