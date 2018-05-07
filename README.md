# kubepf

The Kubernetes apiserver supports [auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/). Operators can use this feature to see whenever a client uses `kubectl exec` to gain access to a container. One issue with this is that operators cannot see what a client actually types during the session which reduces their ability to audit effectively.

kubepf fills in this missing information and allows operators to see exactly what was typed during any shell session. kubepf does this by injecting a small eBPF program on every node which is triggered whenever a TTY is written to. These writes are then submitted to userspace, grouped into sessions and streamed to a central server in the [asciicast](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md) format so that they can be played back in real time.

## Warning 

This project is strictly alpha and kubepf injects code that is run in kernel space. Although this code runs on an in-kernel VM and should be safe, there can be perfomance implications - use at your own risk!

## Quickstart 

- gcloud container clusters create test --image-type UBUNTU
- /usr/src/linux-gcp-headers-4.13.0-1008 and /usr/src/linux-headers-4.13.0-1008-gcp

### Prerequisites

- Kernel version 4.8 or above

### Build

```
make all
```

The generated object file can be viewed using `llvm-objdump`

```
llvm-objdump -S ./bpf/bpf_tty.o
```
