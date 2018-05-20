// curl -X POST -H "Content-Type: application/octet-stream" --data-binary '@filename' http://127.0.0.1:5050/upload

package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dippynark/kubepf/pkg/asciinema"
	"github.com/dippynark/kubepf/pkg/server"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultAddress         = "0.0.0.0"
	defaultPort            = 5050
	defaultDataPath        = "/tmp"
	linuxFilenameSizeLimit = 255
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

	http.HandleFunc("/", healthzHandler)
	http.HandleFunc("/healthz", healthzHandler)
	http.Handle("/list", websocket.Handler(listHandler))
	http.Handle("/upload", websocket.Handler(uploadHandler))

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
	if err != nil {
		glog.Fatalf("ListenAndServe: %s", err)
	}

}

func healthzHandler(rw http.ResponseWriter, r *http.Request) {
	return
}

func listHandler(ws *websocket.Conn) {
	glog.Errorf("list handler invoked")
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
	glog.Errorf("upload handler invoked")
	var files = make(map[string](*os.File))
	var timestamps = make(map[string](int64))

	for {

		var ttyWrite server.TtyWrite

		err := binary.Read(ws, binary.BigEndian, &ttyWrite)
		if err == io.EOF {
			return
		} else if err != nil {
			glog.Fatalf("failed to read from websocket connection: %s", err)
		} else {

			//hash = hostname mount-namespace inode filesystem-identifier
			hash := hash(fmt.Sprintf("%s%d%d", ttyWrite.Hostname, ttyWrite.Inode, ttyWrite.MountNamespaceInum))

			filename := ""
			regex := "[^a-zA-Z0-9-]+"
			reg, err := regexp.Compile(regex)
			if err != nil {
				glog.Fatalf("failed to compile regular expression %s: %s", regex, err)
			}
			for _, attribute := range []string{fmt.Sprintf("%s", ttyWrite.PodNamespace), fmt.Sprintf("%s", ttyWrite.PodName), fmt.Sprintf("%s", ttyWrite.ContainerName)} {
				attribute = reg.ReplaceAllString(attribute, "")
				if len(attribute) > 0 {
					filename = fmt.Sprintf("%s-", attribute)
				}
			}
			if len(filename) == 0 {
				hostname := fmt.Sprintf("%s", ttyWrite.Hostname)
				hostname = reg.ReplaceAllString(hostname, "")
				if len(hostname) > 0 {
					filename = fmt.Sprintf("%s-", hostname)
				}
			}

			filename = fmt.Sprintf("%s/%s.cast", dataPath, strings.Replace(fmt.Sprintf("%s%d", filename, hash), string(0), "", -1))
			if filename[0:1] == "/" {
				if len(filename) > linuxFilenameSizeLimit {
					filename = filename[0:linuxFilenameSizeLimit]
				}
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					glog.Fatalf("could not get working directory: %s", err)
				}
				if len(fmt.Sprintf("%s/%s", cwd, filename)) > linuxFilenameSizeLimit {
					filename = filename[0:linuxFilenameSizeLimit]
				}
			}

			file, ok := files[filename]
			if !ok {

				if _, err = os.Stat(filename); os.IsNotExist(err) {

					file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filename, err)
					}
					defer file.Close()

					timestamp, err := asciinema.Init(&ttyWrite, file)
					if err != nil {
						glog.Fatalf("failed to initialise: %s", err)
					}

					timestamps[filename] = timestamp
				} else if !os.IsNotExist(err) {

					file, err = os.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filename, err)
					}
					defer file.Close()

				}

				files[filename] = file
			}

			timestamp, ok := timestamps[filename]
			if !ok {
				temp, err := os.OpenFile(filename, os.O_RDONLY, 0775)
				if err != nil {
					glog.Fatalf("failed to open file %s to read timestamp: %s", filename, err)
				}
				temp.Seek(0, 0)
				scanner := bufio.NewScanner(temp)
				scanner.Scan()

				var h asciinema.Header
				err = json.Unmarshal([]byte(scanner.Text()), &h)
				if err != nil {
					glog.Fatalf("failed to unmarshal JSON: %s", err)
				}

				timestamp = h.Timestamp
				timestamps[filename] = timestamp

				temp.Close()
			}

			err = asciinema.Append(&ttyWrite, file, timestamp)
			if err != nil {
				glog.Fatalf("failed to write entry: %s", err)
			}
		}
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
