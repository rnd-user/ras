package ras

import (
	"bytes"
	"fmt"
	rfb "github.com/rnd-user/go-vnc"
	"io"
	"time"
)

type vncClient struct {
	*rfb.ClientConn
	readCh  <-chan Message
	writeCh chan<- Message
}

func newVNCClient(readCh <-chan Message, writeCh chan<- Message) (*vncClient, error) {
	cli := &vncClient{
		readCh:  readCh,
		writeCh: writeCh,
	}

	go func() {
		var err error

		// connect
		if err = cli.connect(); err != nil {
			close(writeCh)
			return
		}

		// send pixel format & supported encodings
		msgs := []rfb.ClientMessage{
			&rfb.SetPixelFormatMsg{
				ID: rfb.SetPixelFormatMID,
				RFBPixelFormat: rfb.RFBPixelFormat{
					BPP:        16,
					Depth:      16,
					BigEndian:  0,
					TrueColor:  1,
					RedMax:     31,
					GreenMax:   63,
					BlueMax:    31,
					RedShift:   0,
					GreenShift: 5,
					BlueShift:  11,
				},
			},
			&rfb.SetEncodingsMsg{
				ID: rfb.SetEncodingsMID,
				Encodings: []rfb.Encoding{
					&rfb.CopyRectEncoding{},
					&rfb.HextileEncoding{},
					&rfb.CursorPseudoEncoding{},
					&rfb.DesktopSizePseudoEncoding{},
				},
			},
		}
		for _, msg := range msgs {
			if err = cli.SendMsg(msg); err != nil {
				close(writeCh)
				return
			}
		}

		// resize client
		writeCh <- &ResizeMsg{ResizeMID, cli.FrameBufferWidth, cli.FrameBufferHeight}

		// start worker threads
		go cli.startReader()
		go cli.startWriter()
		return
	}()

	return cli, nil
}

func (cli *vncClient) connect() error {
	// read connect params
	var binMsg *BinaryMsg
	if msg, ok := <-cli.readCh; !ok {
		return fmt.Errorf("client closed unexpectedly")
	} else if binMsg, ok = msg.(*BinaryMsg); !ok {
		return fmt.Errorf("unexpected message %d (expected BinaryMsg)", msg.ID())
	}

	// parse address & password
	r := bytes.NewBuffer(binMsg.Bytes)
	var address, password string
	params := []*string{&address, &password}
	for _, param := range params {
		var length16 uint16
		if err := readFixedSize(r, &length16); err != nil {
			return err
		}
		buf := make([]uint8, length16)
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		*param = string(buf)
	}

	// connect & handshake
	cfg := &rfb.ClientConnConfig{
		Address:        address,
		Auth:           []rfb.ClientAuth{&rfb.VNCAuth{password}},
		Exclusive:      false,
		ServerMessages: make(map[rfb.MessageID]rfb.ServerMessage, 4),
	}
	var err error
	if cli.ClientConn, err = rfb.NewClientConn(cfg, nil); err != nil {
		return err
	} else if err = cli.Handshake(); err != nil {
		cli.Close()
		return err
	}

	return nil
}

func (cli *vncClient) startReader() {
	var err error
	var rfbMsg rfb.ClientMessage
	ticker := time.NewTicker(33 * time.Millisecond) // 30fps
	stop := func() {
		ticker.Stop()
		cli.Close()
	}
	stopNDrain := func() {
		stop()
		for range cli.readCh {
		}
	}

Loop:
	for {
		select {
		case msg, ok := <-cli.readCh:
			if !ok {
				stop()
				break Loop
			} else {
				rfbMsg, err = cli.translateMsg(msg)
			}
		case <-ticker.C:
			rfbMsg = &rfb.FramebufferUpdateRequestMsg{
				ID:          rfb.FramebufferUpdateRequestMID,
				Incremental: 1,
				X:           0,
				Y:           0,
				Width:       cli.FrameBufferWidth,
				Height:      cli.FrameBufferHeight,
			}
		}
		if err != nil {
			stopNDrain()
			break
		} else if err = cli.SendMsg(rfbMsg); err != nil {
			stopNDrain()
			break
		}
	}
}

func (cli *vncClient) startWriter() {
	for {
		if msgs, err := cli.prepareMsgs(); err != nil {
			close(cli.writeCh)
			break
		} else {
			for _, msg := range msgs {
				cli.writeCh <- msg
			}
		}
	}
}

func (cli *vncClient) translateMsg(msg Message) (rfbMsg rfb.ClientMessage, err error) {
	switch c := msg.(type) {
	case *KeyboardEventMsg:
		rfbMsg = &rfb.KeyEventMsg{
			ID:       rfb.KeyEventMID,
			DownFlag: c.DownFlag,
			Key:      c.Key,
		}
	case *MouseEventMsg:
		// only keep left, mid & right buttons
		var bMask uint8 = uint8(0x7 & c.Buttons)
		// swap bit 1 & 2 (mid/right buttons)
		tmp := ((bMask >> 1) ^ (bMask >> 2)) & 1
		bMask ^= (tmp << 1) | (tmp << 2)
		// create rfb msg
		rfbMsg = &rfb.PointerEventMsg{rfb.PointerEventMID, bMask, c.X, c.Y}
	}
	return rfbMsg, nil
}

func (cli *vncClient) prepareMsgs() (msgs []Message, err error) {
	var msg rfb.ServerMessage
	if msg, err = cli.ReceiveMsg(); err != nil {
		return
	}

Outer:
	switch m := msg.(type) {
	case *rfb.FramebufferUpdateMsg:
		var imgBuf []byte
		msgs = make([]Message, 0, len(m.Rectangles))

		for _, rect := range m.Rectangles {
			switch enc := rect.Encoding.(type) {
			case *rfb.RawEncoding:
				imgBuf, err = enc.PNG(&rect)
			case *rfb.HextileEncoding:
				imgBuf, err = enc.PNG(&rect)

			case *rfb.CopyRectEncoding:
				msgs = append(msgs, &CopyMsg{CopyMID, rect.X, rect.Y, rect.Width, rect.Height, enc.SX, enc.SY})
			case *rfb.DesktopSizePseudoEncoding:
				cli.FrameBufferWidth = rect.Width
				cli.FrameBufferHeight = rect.Height
				msgs = append(msgs, &ResizeMsg{ResizeMID, rect.Width, rect.Height})
			case *rfb.CursorPseudoEncoding:
				if imgBuf, err = enc.PNG(&rect); err == nil {
					msgs = append(msgs, &CursorMsg{CursorMID, rect.X, rect.Y, imgBuf})
					imgBuf = nil
				}
			}

			if err != nil {
				break Outer
			} else if imgBuf != nil {
				msgs = append(msgs, &PngMsg{
					PngMID,
					rect.X, rect.Y, rect.Width, rect.Height,
					imgBuf,
				})
				imgBuf = nil
			}
		}

	case *rfb.ServerCutTextMsg:
		msgs = []Message{&TextMsg{TextMID, m.Text}}
	case *rfb.SetColorMapEntriesMsg:
		err = cli.PixelFormat().ColorMap.UpdateColorMap(m.FirstColor, m.Colors)
		msgs = []Message{}
	case *rfb.BellMsg:
		msgs = []Message{}

	default:
		err = fmt.Errorf("unsupported server message %v", msg.ID())
	}

	return
}
