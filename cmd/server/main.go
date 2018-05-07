// curl -X POST -H "Content-Type: application/octet-stream" --data-binary '@filename' http://127.0.0.1:5050/upload

package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dippynark/kubepf/pkg/asciinema"
	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultAddress = "0.0.0.0"
	defaultPort    = 5050
)

func uploadHandler(ws *websocket.Conn) {

	var files = make(map[string](*os.File))

	for {

		var ttyWrite kubepf.TtyWrite

		err := binary.Read(ws, binary.BigEndian, &ttyWrite)
		if err == io.EOF {
			return
		} else if err != nil {
			glog.Fatalf("failed to read from websocket connection: %s", err)
		} else {

			//hash - hostname mount-namespace filesystem-identifier
			hasher := sha1.New()
			hasher.Write([]byte(fmt.Sprintf("%s%s", ttyWrite.Hostname, ttyWrite.Inode)))
			sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
			filname := fmt.Sprintf("%s.cast", sha)

			file, ok := files[sha]
			if !ok {

				if _, err = os.Stat(filname); os.IsNotExist(err) {

					file, err = os.OpenFile(filname, os.O_CREATE|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filname, err)
					}
					defer file.Close()

					err = asciinema.Init(&ttyWrite, file)
					if err != nil {
						glog.Fatalf("failed to initialise: %s", err)
					}
				} else if !os.IsNotExist(err) {

					file, err = os.OpenFile(filname, os.O_APPEND|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filname, err)
					}
					defer file.Close()

				}

				files[sha] = file
			}

			err = asciinema.Append(&ttyWrite, file)
			if err != nil {
				glog.Fatalf("failed to write entry: %s", err)
			}
		}
	}
}

func main() {

	address := *flag.String("address", defaultAddress, "address to serve on")
	port := *flag.Int("port", defaultPort, "port to serve on")
	flag.Parse()

	http.Handle("/upload", websocket.Handler(uploadHandler))
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
	if err != nil {
		glog.Fatalf("ListenAndServe: %s", err)
	}

}
