package ras

import (
	"io"
)

type MessageID uint16

type Message interface {
	ID() MessageID
}

type Receiver interface {
	Receive(io.Reader) error
}

type Sender interface {
	Send(io.Writer) error
}

const (
	BinaryMID MessageID = iota
	TextMID
)

type BinaryMsg struct {
	MID   MessageID
	Size  uint32
	Bytes []byte
}

func (msg *BinaryMsg) ID() MessageID {
	return BinaryMID
}

func (msg *BinaryMsg) Send(w io.Writer) error {
	return writeFixedSize(w, msg)
}

func (msg *BinaryMsg) Receive(r io.Reader) error {
	if err := readFixedSize(r, &msg.Size); err != nil {
		return err
	}
	msg.Bytes = make([]byte, msg.Size)
	if _, err := io.ReadFull(r, msg.Bytes); err != nil {
		return err
	}
	return nil
}

type TextMsg struct {
	MID  MessageID
	Text string
}

func (msg *TextMsg) ID() MessageID {
	return TextMID
}

func (msg *TextMsg) Send(w io.Writer) error {
	return (&BinaryMsg{
		MID:   TextMID,
		Size:  uint32(len(msg.Text)),
		Bytes: []byte(msg.Text),
	}).Send(w)
}

func (msg *TextMsg) Receive(r io.Reader) error {
	binMsg := &BinaryMsg{}
	if err := binMsg.Receive(r); err != nil {
		return err
	}
	msg.MID = TextMID
	msg.Text = string(binMsg.Bytes)
	return nil
}
