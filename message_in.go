package ras

import (
	"io"
)

const (
	ProtocolMID MessageID = 1000 + 2*iota
	KeyboardEventMID
	MouseEventMID
)

// len:protocol
type ProtocolMsg struct {
	Protocol string
}

func (*ProtocolMsg) ID() MessageID {
	return ProtocolMID
}

func (msg *ProtocolMsg) Receive(r io.Reader) error {
	var length8 uint8
	if err := readFixedSize(r, &length8); err != nil {
		return err
	}
	buf := make([]byte, length8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	msg.Protocol = string(buf)
	return nil
}

type KeyboardEventMsg struct {
	DownFlag uint8
	Key      uint32
}

func (*KeyboardEventMsg) ID() MessageID {
	return KeyboardEventMID
}

func (msg *KeyboardEventMsg) Receive(r io.Reader) error {
	return readFixedSize(r, msg)
}

type MouseEventMsg struct {
	Buttons uint16
	X       uint16
	Y       uint16
}

func (*MouseEventMsg) ID() MessageID {
	return MouseEventMID
}

func (msg *MouseEventMsg) Receive(r io.Reader) error {
	return readFixedSize(r, msg)
}
