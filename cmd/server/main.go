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
	"strconv"
	"time"

	"github.com/dippynark/kubecast/pkg/asciinema"
	"github.com/dippynark/kubecast/pkg/server"
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

	var labels = make(map[uint32]([]string))

	for {

		message := ""
		files, err := filepath.Glob(dataPath + "/*.cast")
		if err != nil {
			glog.Fatalf("could not list files: %s", err)
		}
		for _, file := range files {

			basename := filepath.Base(file)
			hashString := basename[0 : len(basename)-len(filepath.Ext(basename))]
			hashInt, err := strconv.ParseInt(hashString, 10, 64)
			if err != nil {
				glog.Fatalf("failed to parse %s to int: %s", hashString, err)
			}
			hash := uint32(hashInt)

			labelsArray, ok := labels[hash]
			if !ok {

				labelsFilename := fmt.Sprintf("%s%s", file, ".labels")
				labelsFile, err := os.OpenFile(labelsFilename, os.O_RDONLY, 0775)
				if err != nil {
					glog.Fatalf("failed to open file %s: %s", labelsFilename, err)
				}
				scanner := bufio.NewScanner(labelsFile)
				for scanner.Scan() {
					labels[hash] = append(labels[hash], scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					glog.Fatalf("failed to read file %s: %s", labelsFilename, err)
				}
				labelsFile.Close()

				labelsArray = labels[hash]
			}

			labelsString := ""
			for _, label := range labelsArray {
				labelsString = fmt.Sprintf("%s%s.", labelsString, label)
			}

			message += (file + "\n" + labelsString + "\n")
		}

		n, err := ws.Write([]byte(message))
		if err != nil {
			glog.Errorf("failed to write message: %s", err)
			return
		}
		if n != len(message) {
			glog.Errorf("could only write %d out of %d bytes", n, len(message))
			return
		}

		time.Sleep(time.Second)
	}

}

func uploadHandler(ws *websocket.Conn) {
	glog.Errorf("upload handler invoked")
	var files = make(map[uint32](*os.File))
	var timestamps = make(map[uint32](int64))

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
			labels := []string{
				fmt.Sprintf("%s", ttyWrite.PodNamespace),
				fmt.Sprintf("%s", ttyWrite.PodName),
				fmt.Sprintf("%s", ttyWrite.ContainerName),
				fmt.Sprintf("%s", ttyWrite.Hostname)}

			reg, err := regexp.Compile(regex)
			if err != nil {
				glog.Fatalf("failed to compile regular expression %s: %s", regex, err)
			}
			for i, label := range labels {
				labels[i] = reg.ReplaceAllString(label, "")
			}

			labelsFilename := fmt.Sprintf("%s/%d.cast.labels", dataPath, hash)
			if _, err = os.Stat(labelsFilename); os.IsNotExist(err) {

				labelFile, err := os.OpenFile(labelsFilename, os.O_CREATE|os.O_RDWR, 0775)
				if err != nil {
					glog.Fatalf("failed to open file %s: %s", labelsFilename, err)
				}
				labelsString := ""
				for _, label := range labels {
					labelsString = fmt.Sprintf("%s%s\n", labelsString, label)
				}

				n, err := labelFile.Write([]byte(labelsString))
				if err != nil {
					glog.Errorf("failed to write labels: %s", err)
					return
				}
				if n != len(labelsString) {
					glog.Errorf("could only write %d out of %d bytes", n, len(labelsString))
					return
				}
				labelFile.Close()

			} else if err != nil {
				glog.Fatalf("failed to stat file %s: %s", labelsFilename, err)
			}

			filename = fmt.Sprintf("%s/%d.cast", dataPath, hash)
			file, ok := files[hash]
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

					timestamps[hash] = timestamp
				} else if err != nil {

					glog.Fatalf("failed to stat file %s: %s", filename, err)

				} else {

					file, err = os.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0775)
					if err != nil {
						glog.Fatalf("failed to open file %s: %s", filename, err)
					}
					defer file.Close()

				}

				files[hash] = file
			}

			timestamp, ok := timestamps[hash]
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
				timestamps[hash] = timestamp

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
