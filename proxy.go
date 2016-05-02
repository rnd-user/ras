package ras

import (
	"fmt"
)

type Client interface {
	Channels() (<-chan Message, chan<- Message)
}

func Serve(client Client) error {
	// select protocol
	readCh, writeCh := client.Channels()
	var protMsg *ProtocolMsg
	if msg, ok := <-readCh; !ok {
		return fmt.Errorf("client closed unexpectedly")
	} else if protMsg, ok = msg.(*ProtocolMsg); !ok {
		return fmt.Errorf("unexpected message %d (expected ProtocolMsg)", msg.ID())
	} else if err := initServer(protMsg.Protocol, readCh, writeCh); err != nil {
		return err
	}
	return nil
}
