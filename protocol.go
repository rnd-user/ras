package ras

import (
	"fmt"
)

func initServer(protocol string, readCh <-chan Message, writeCh chan<- Message) (err error) {
	switch protocol {
	case "vnc":
		_, err = newVNCClient(readCh, writeCh)
	default:
		err = fmt.Errorf("unsupported protocol %s", protocol)
	}
	return err
}
