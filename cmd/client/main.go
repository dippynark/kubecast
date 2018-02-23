package main

import (
	"flag"
	"fmt"
	"os"
	"unsafe"
	"net/http"
	"bytes"

	"github.com/dippynark/kubepf/pkg/kubepf"
)

/*
#include "../../bpf/bpf_tty.h"
*/
import "C"

const (
	sessionIDHTTPHeader  = "X-Session-ID"
	defaultServerAddress = "127.0.0.1"
	defaultPort          = 5050
)

type ttyWriteTracer struct {
	lastTimestamp uint64
}

type ttyWrite struct {
	Count    uint32
	Buffer   string
	SessionID uint32
	Timestamp uint64
}

func (t *ttyWriteTracer) Send(ttyWrite ttyWrite, server string) {

	payload := bytes.NewBufferString(ttyWrite.Buffer[0:ttyWrite.Count])
  req, err := http.NewRequest("POST", server, payload)
	req.Header.Set(sessionIDHTTPHeader, fmt.Sprintf("%d", ttyWrite.SessionID))
	req.Header.Set("Content-Type", "binary/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending data")
	}
	defer resp.Body.Close()

	if t.lastTimestamp > ttyWrite.Timestamp {
		fmt.Printf("ERROR: late event!\n")
		os.Exit(1)
	}

	t.lastTimestamp = ttyWrite.Timestamp
}

func main() {

	address := flag.String("server", defaultServerAddress, "address of server")
	port := flag.Int("port", defaultPort, "port to connect to")
	flag.Parse()

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	err := kubepf.New(channel, lostChannel)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	t := &ttyWriteTracer{}

	for {
		select {
		case ttyWrite, ok := <-channel:
			if !ok {
				return // see explanation above
			}
			ttyWriteGo := ttyWriteToGo(&ttyWrite)
			t.Send(ttyWriteGo, fmt.Sprintf("http://%s:%d/upload", *address, *port))
		case lost, ok := <-lostChannel:
			if !ok {
				return // see explanation above
			}
			fmt.Printf("data lost: %#v", lost)
		}
	}

}

func ttyWriteToGo(data *[]byte) (ret ttyWrite) {

	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))

	ret.Count = uint32(ttyWrite.count)
	ret.Buffer = C.GoString(&ttyWrite.buf[0])
	ret.SessionID = uint32(ttyWrite.sessionid)
	ret.Timestamp = uint64(ttyWrite.timestamp)

	return
}
