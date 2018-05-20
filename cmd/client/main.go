package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"time"

	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultServerAddress = "localhost"
	defaultPort          = 5050
)

var serverAddressFlag = flag.String("server", defaultServerAddress, "server address")
var portFlag = flag.Int("port", defaultPort, "server port")

func main() {

	flag.Parse()

	serverAddress := *serverAddressFlag
	port := *portFlag

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	err := kubepf.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfully")

	for {

		// connect to server
		ws, err := websocket.Dial(fmt.Sprintf("ws://%s:%d/upload", serverAddress, port), "", fmt.Sprintf("http://%s/", serverAddress))
		if err != nil {
			glog.Errorf("failed to connect to server: %s", err)
			time.Sleep(1)
			continue
		}
	L:
		for {

			select {
			case ttyWrite, ok := <-channel:

				if !ok {
					glog.Fatal("channel closed")
				}

				ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)

				err = binary.Write(ws, binary.BigEndian, ttyWriteGo)
				if err != nil {
					glog.Errorf("failed to write to websocket connection: %s", err)
					ws.Close()
					break L
				}

			case lost, ok := <-lostChannel:
				if !ok {
					glog.Fatal("lost channel closed")
				}
				glog.Errorf("data lost: %#v", lost)
			}
		}
	}
}
