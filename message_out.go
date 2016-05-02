package ras

import (
	"io"
)

const (
	ResizeMID MessageID = 1001 + 2*iota
	PngMID
	CopyMID
	CursorMID
)

type ResizeMsg struct {
	MID           MessageID
	Width, Height uint16
}

func (*ResizeMsg) ID() MessageID {
	return ResizeMID
}

func (msg *ResizeMsg) Send(w io.Writer) error {
	return writeFixedSize(w, msg)
}

type PngMsg struct {
	MID                 MessageID
	X, Y, Width, Height uint16
	Img                 []byte
}

func (*PngMsg) ID() MessageID {
	return PngMID
}

func (msg *PngMsg) Send(w io.Writer) (err error) {
	data := []uint16{uint16(msg.MID), msg.X, msg.Y, msg.Width, msg.Height}
	if err = writeFixedSize(w, data); err != nil {
		return err
	} else if err = writeFixedSize(w, uint32(len(msg.Img))); err != nil {
		return err
	} else if _, err = w.Write(msg.Img); err != nil {
		return err
	}
	return nil
}

type CopyMsg struct {
	MID                           MessageID
	DX, DY, Width, Height, SX, SY uint16
}

func (*CopyMsg) ID() MessageID {
	return CopyMID
}

func (msg *CopyMsg) Send(w io.Writer) error {
	return writeFixedSize(w, msg)
}

type CursorMsg struct {
	MID  MessageID
	X, Y uint16
	Img  []byte
}

func (*CursorMsg) ID() MessageID {
	return CursorMID
}

func (msg *CursorMsg) Send(w io.Writer) (err error) {
	data := []uint16{uint16(msg.MID), msg.X, msg.Y}
	if err = writeFixedSize(w, data); err != nil {
		return err
	} else if err = writeFixedSize(w, uint32(len(msg.Img))); err != nil {
		return err
	} else if _, err = w.Write(msg.Img); err != nil {
		return err
	}
	return nil
}
