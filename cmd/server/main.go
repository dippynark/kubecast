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
	"path/filepath"
	"time"

	"github.com/dippynark/kubepf/pkg/asciinema"
	"github.com/dippynark/kubepf/pkg/server"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultAddress  = "0.0.0.0"
	defaultPort     = 5050
	defaultDataPath = "/tmp"
)

var addressFlag = flag.String("address", defaultAddress, "address to serve on")
var portFlag = flag.Int("port", defaultPort, "port to serve on")
var dataPathFlag = flag.String("data-path", defaultDataPath, "directory to store data")

var dataPath string

func main() {

	flag.Parse()

	address := *addressFlag
	port := *portFlag
	dataPath = *dataPathFlag

	stat, err := os.Stat(dataPath)
	if err != nil {
		glog.Fatalf("could not stat path %s: %s", dataPath, err)
	}
	if !stat.IsDir() {
		glog.Fatalf("%s is not a directory", dataPath)
	}

	http.Handle("/list", websocket.Handler(listHandler))
	http.Handle("/upload", websocket.Handler(uploadHandler))

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
	if err != nil {
		glog.Fatalf("ListenAndServe: %s", err)
	}

}

func listHandler(ws *websocket.Conn) {
	glog.Errorf("Connection made")
	for {

		message := ""
		files, err := filepath.Glob(dataPath + "/*.cast")
		if err != nil {
			glog.Fatalf("could not list files: %s", err)
		}
		for _, file := range files {
			message += (file + "\n")
		}

		n, err := ws.Write([]byte(message))
		if n != len(message) {
			glog.Errorf("could only write %d out of %d bytes", n, len(message))
			return
		}
		if err != nil {
			glog.Errorf("failed to write message: %s", err)
			return
		}
		time.Sleep(time.Second)
	}

}

func uploadHandler(ws *websocket.Conn) {

	var files = make(map[string](*os.File))

	for {

		var ttyWrite server.TtyWrite

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
			filename := fmt.Sprintf("%s/%s.cast", dataPath, sha)

			file, ok := files[sha]
			if !ok {

				if _, err = os.Stat(filename); os.IsNotExist(err) {

					file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filename, err)
					}
					defer file.Close()

					err = asciinema.Init(&ttyWrite, file)
					if err != nil {
						glog.Fatalf("failed to initialise: %s", err)
					}
				} else if !os.IsNotExist(err) {

					file, err = os.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filename, err)
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
