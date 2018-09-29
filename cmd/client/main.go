package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dippynark/kubecast/pkg/kubecast"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/golang/glog"
	"golang.org/x/net/websocket"
)

const (
	defaultServerAddress = "localhost"
	defaultPort          = 5050

	kubernetesPodNameKey       = "io.kubernetes.pod.name"
	kubernetesPodNamespaceKey  = "io.kubernetes.pod.namespace"
	kubernetesContainerNameKey = "io.kubernetes.container.name"
	kubernetesPodUIDKey        = "io.kubernetes.pod.uid"
)

var serverAddressFlag = flag.String("server", defaultServerAddress, "server address")
var portFlag = flag.Int("port", defaultPort, "server port")

func main() {

	flag.Parse()

	serverAddress := *serverAddressFlag
	port := *portFlag

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	err := kubecast.New(channel, lostChannel)
	if err != nil {
		glog.Fatalf("failed to load BPF module: %s", err)
	}
	glog.Info("loaded BPF program successfully")

	cli, err := client.NewEnvClient()
	if err != nil {
		glog.Fatalf("failed to create Docker client: %s", err)
	}

	mountNamespaceToContainerLabels := refresh(cli)

OUTER:
	for {

		// connect to server
		glog.Info("attempting connection to server...")
		connectionChannel := make(chan *websocket.Conn)
		go func() {
			ws, err := websocket.Dial(fmt.Sprintf("ws://%s:%d/upload", serverAddress, port), "", fmt.Sprintf("http://%s/", serverAddress))
			if err != nil {
				glog.Errorf("failed to connect to server: %s", err)
				return
			}
			connectionChannel <- ws
		}()

		var ws *websocket.Conn
		select {
		case ws = <-connectionChannel:
		case <-time.After(3 * time.Second):
			glog.Errorf("timeout connecting to server")
			continue OUTER
		}
		glog.Info("connection successful")

		for {

			select {
			case ttyWrite, ok := <-channel:

				if !ok {
					glog.Fatal("channel closed")
				}

				ttyWriteGo := kubecast.TtyWriteToGo(&ttyWrite)
				containerLabels, ok := mountNamespaceToContainerLabels[fmt.Sprintf("%d", ttyWriteGo.MountNamespaceInum)]
				if !ok {
					mountNamespaceToContainerLabels = refresh(cli)
					containerLabels, ok = mountNamespaceToContainerLabels[fmt.Sprintf("%d", ttyWriteGo.MountNamespaceInum)]
				}

				copy(ttyWriteGo.ContainerName[:], containerLabels[kubernetesContainerNameKey])
				copy(ttyWriteGo.PodName[:], containerLabels[kubernetesPodNameKey])
				copy(ttyWriteGo.PodNamespace[:], containerLabels[kubernetesPodNamespaceKey])
				copy(ttyWriteGo.PodUID[:], containerLabels[kubernetesPodUIDKey])

				//glog.Errorf("%s %s %s %s", containerLabels[kubernetesContainerNameKey], containerLabels[kubernetesPodNameKey], containerLabels[kubernetesPodNamespaceKey], containerLabels[kubernetesPodUIDKey])
				//glog.Errorf("test NS: %d %#v", ttyWriteGo.MountNamespaceInum, containerLabels)
				//glog.Errorf("MountNamespaceInum: %s", ttyWriteGo.MountNamespaceInum)

				err = binary.Write(ws, binary.BigEndian, ttyWriteGo)
				if err != nil {
					glog.Errorf("failed to write to websocket connection: %s", err)
					ws.Close()
					continue OUTER
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

func refresh(cli *client.Client) map[string](map[string]string) {

	mountNamespaceToContainerLabels := make(map[string](map[string]string))

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	//glog.Errorf("Debug: %#v", containers)

	for _, container := range containers {
		//fmt.Printf("%s %s\n", container.ID[:10], container.)
		ContainerJSON, err := cli.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			glog.Errorf("failed to inspect container with ID %s: %s", container.ID, err)
			continue
		}

		//glog.Errorf("NS: %#v", mountNamespace, ContainerJSON)

		pid := 0
		if ContainerJSON.ContainerJSONBase != nil {
			if ContainerJSON.ContainerJSONBase.State != nil {
				pid = ContainerJSON.ContainerJSONBase.State.Pid
			}
		}

		if pid != 0 {

			mountNamespaceFile, err := os.Readlink(fmt.Sprintf("/proc/%d/ns/mnt", pid))
			if err != nil {
				glog.Errorf("failed to retrieve namespace for PID %d: %s", pid, err)
				continue
			}

			mountNamespace := strings.Split(strings.Split(mountNamespaceFile, "[")[1], "]")[0]
			mountNamespaceToContainerLabels[mountNamespace] = ContainerJSON.Config.Labels

			//glog.Errorf("NS: %s %#v", mountNamespace, ContainerJSON.Config.Labels)

		}
	}

	return mountNamespaceToContainerLabels
}
