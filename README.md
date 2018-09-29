# kubecast

The Kubernetes apiserver supports [auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/). Operators can use this feature to see whenever a client uses `kubectl exec` to gain access to a container. One issue with this is that operators cannot see what a client actually types during the session which reduces their ability to audit effectively.

kubecast fills in this missing information and allows operators to see exactly what was typed during any shell session. kubecast does this by injecting a small eBPF program on every node which is triggered whenever a TTY is written to. These writes are then submitted to userspace, grouped into sessions and streamed to a central server in the [asciicast](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md) format so that they can be played back in real time.

## Warning 

This project is strictly alpha and kubecast injects code that is run in kernel space. Although this code runs on an in-kernel VM and should be safe, there can be performance implications - use at your own risk!

### Prerequisites

- Kernel version 4.8 or above

### Build

```
apt-cache search linux-image
apt-get install linux-image-4.15.0-1019-gcp
# reboot
apt-get install linux-headers-`uname -r`
make
make docker_push
```

### Run

```
gcloud container clusters create kubecast --image-type UBUNTU
```

### Debug

The generated object file can be viewed using `llvm-objdump`

```
llvm-objdump -S ./bpf/bpf_tty.o
```
