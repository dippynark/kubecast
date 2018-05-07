package main

import (
	"encoding/binary"
	"flag"
	"fmt"

	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultServerAddress = "localhost"
	defaultPort          = 5050
)

func main() {

	serverAddress := *flag.String("server", defaultServerAddress, "server address")
	port := *flag.Int("port", defaultPort, "server port")
	flag.Parse()

	// connect to server
	ws, err := websocket.Dial(fmt.Sprintf("ws://%s:%d/upload", serverAddress, port), "", fmt.Sprintf("http://%s/", serverAddress))
	if err != nil {
		glog.Fatalf("failed to connect to server: %s", err)
	}

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	err = kubepf.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfully")

	for {
		select {
		case ttyWrite, ok := <-channel:

			if !ok {
				glog.Fatal("channel closed")
			}

			ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)

			err = binary.Write(ws, binary.BigEndian, ttyWriteGo)
			if err != nil {
				glog.Fatalf("failed to write to websocket connection: %s", err)
			}

		case lost, ok := <-lostChannel:
			if !ok {
				glog.Fatal("lost channel closed")
			}
			glog.Errorf("data lost: %#v", lost)
		}
	}
}
