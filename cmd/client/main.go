package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dippynark/kubepf/pkg/asciinema"
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

	err := kubepf.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfully")

	files := make(map[uint64](*os.File))

	for {
		select {
		case ttyWrite, ok := <-channel:

			if !ok {
				glog.Fatal("channel closed")
			}
			ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)

			file, ok := files[ttyWriteGo.Inode]
			if !ok {
				file, err = os.OpenFile(fmt.Sprintf("%d.json", ttyWriteGo.Inode), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0775)
				if err != nil {
					glog.Fatalf("failed to open file %s", fmt.Sprintf("%d.json", ttyWriteGo.Inode))
				}
				files[ttyWriteGo.Inode] = file
				defer file.Close()

				err = asciinema.Init(&ttyWriteGo, file)
				if err != nil {
					glog.Fatalf("failed to initialise: %s", err)
				}
			}

			err = asciinema.Append(&ttyWriteGo, file)
			if err != nil {
				glog.Fatalf("failed to write entry: %s", err)
			}

		case lost, ok := <-lostChannel:
			if !ok {
				glog.Fatal("lost channel closed")
			}
			glog.Errorf("data lost: %#v", lost)
		}
	}
}
