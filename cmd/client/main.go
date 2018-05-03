package main

import (
	"flag"
	"fmt"
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
)

const (
	defaultServerAddress = "127.0.0.1"
	defaultPort          = 5050
)

func main() {

	flag.Parse()

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	ttyWriteTracer, err := kubepf.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfully")

	go func() {
		for {
			select {
			case ttyWrite, ok := <-channel:
				if !ok {
					glog.Fatal("channel closed")
				}
				ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)
				fmt.Printf("%s", ttyWriteGo.Buffer[0:ttyWriteGo.Count])
			case lost, ok := <-lostChannel:
				if !ok {
					glog.Fatal("lost channel closed")
				}
				glog.Error("data lost: %#v", lost)
			}
		}
	}()

	for {
		fmt.Print("Enter SID: ")
                reader := bufio.NewReader(os.Stdin)
		sidString, _ := reader.ReadString('\n')
		sidString = strings.TrimSuffix(sidString, "\n")
		sid, err := strconv.Atoi(sidString)
		if err != nil {
			glog.Errorf("could not convert SID %s to integer: %s", sidString, err)
			continue
		}
                fmt.Printf("SID: %d\n", sid)
		err = ttyWriteTracer.SetActiveSID(sid)
		if err != nil {
			glog.Errorf("failed to set active SID: %s", err)
			continue
		}
	}

}





