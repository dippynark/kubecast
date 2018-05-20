package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"time"
	"path/filepath"
	"strings"

	"github.com/dippynark/kubepf/pkg/kubepf"
	"github.com/golang/glog"
	"github.com/moby/moby/client"
	"golang.org/x/net/websocket"
)

const (
	defaultServerAddress = "localhost"
	defaultPort          = 5050

	kubernetesPodNameKey = "io.kubernetes.pod.name"
	kubernetesPodNamespaceKey = "io.kubernetes.pod.namespace"
	kubernetesContainerNameKey = "io.kubernetes.container.name"
	kubernetesPodUIDKey = "io.kubernetes.pod.uid"
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

	cli, err := client.NewEnvClient()
	if err != nil {
		glog.Fatalf("failed to create Docker client: %s", err)
	}

	mountNamespaceToContainerLabels := refresh(cli)

	for {

	L:
		// connect to server
		ws, err := websocket.Dial(fmt.Sprintf("ws://%s:%d/upload", serverAddress, port), "", fmt.Sprintf("http://%s/", serverAddress))
		if err != nil {
			glog.Errorf("failed to connect to server: %s", err)
			time.Sleep(1)
			continue
		}

		for {

			select {
			case ttyWrite, ok := <-channel:

				if !ok {
					glog.Fatal("channel closed")
				}

				ttyWriteGo := kubepf.TtyWriteToGo(&ttyWrite)
				labels, ok := mountNamespaceToContainerLabels[ttyWriteGo.MountNamespaceInum]
				if !ok {
					mountNamespaceToContainerLabels := refresh(cli)
					labels, ok = mountNamespaceToContainerLabels[ttyWriteGo.MountNamespaceInum]
				}

				copy(ttyWriteGo.ContainerName[:], ttyWriteGo[kubernetesContainerNameKey])
				copy(ttyWriteGo.PodName[:], ttyWriteGo[kubernetesPodNameKey])
				copy(ttyWriteGo.PodNamespaceKey[:], ttyWriteGo[kubernetesPodNamespaceKey])
				copy(ttyWriteGo.PodUIDKey[:], ttyWriteGo[kubernetesPodUIDKey])

				err = binary.Write(ws, binary.BigEndian, ttyWriteGo)
				if err != nil {
					glog.Errorf("failed to write to websocket connection: %s", err)
					ws.Close()
					goto L
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

func refresh(cli *client.Client) map[uint64](map[string]string) {

	mountNamespaceToContainerLabels := make(map[uint64](map[string]string))
	
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		//fmt.Printf("%s %s\n", container.ID[:10], container.)
		json, err = cli.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			glog.Errorf("failed to inspect container with ID %s: %s", container.ID, err)
			continue
		}
		
		pid := 0
		if _, ok := json.ContainerJSONBase; ok != nil {
			if _,ok = json.ContainerJSONBase.State; ok != nil {
				pid = json.ContainerJSONBase.State.Pid
			}
		}

		if pid != 0 {

			mountNamespaceFile, err = filepath.EvalSymlinks(fmt.Sprintf("/proc/%d/ns/mnt", pid))
			if err != nil {
				glog.Errorf("failed to retrieve namespace for PID %d: %s", pid, err)
				continue
			}

			mountNamespace := strings.Split(strings.Split(mountNamespaceFile, "[")[1], "]")[0]
			mountNamespaceToContainerLabels[uint64(mountNamespace)] = container.Config.Labels

		}		
	}

	return mountNamespaceToContainerLabels
}