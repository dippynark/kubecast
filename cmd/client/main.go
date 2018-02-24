package main

import (
	"flag"
	"fmt"

	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
)

const (
	defaultServerAddress = "127.0.0.1"
	defaultPort          = 5050
)

func main() {

	address := flag.String("server", defaultServerAddress, "address of server")
	port := flag.Int("port", defaultPort, "port to connect to")
	flag.Parse()

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	err := kubepf.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfullly")

	t := &kubepf.TtyWriteTracer{}

	for {
		select {
		case ttyWrite, ok := <-channel:
			if !ok {
				glog.Fatal("channel closed")
			}
			ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)
			t.Upload(ttyWriteGo, fmt.Sprintf("http://%s:%d/upload", *address, *port))
		case lost, ok := <-lostChannel:
			if !ok {
				glog.Fatal("lost channel closed")
			}
			glog.Error("data lost: %#v", lost)
		}
	}

}
