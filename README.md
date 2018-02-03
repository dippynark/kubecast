# kubepf

Have you ever wanted to monitor shell activity on your cluster? When running a Kubernetes cluster, we can use [advanced auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/) to log when a user uses `kubectl exec` to interact with cluster workloads, but we cannot see the actual commands that were run during a session. This inability reduces our ability to audit services effectively.

kubepf is designed to fix this by logging container session activity and streaming the contents to a central database. The way kubepf works is by deploying a DaemonSet to install a set of eBPF programs and a collection Pod on each cluster node. These eBPF programs are inserted on particular instructions in the kernel and userspace and are triggered by certain events. These events allow us to collect interesting information such as function arguments. These arguments could include the text written to a tty or the name of an executed program. 

Furthermore, the information collected from these events can be grouped per session. By passing this information to the collection Pods and then forwarding the data to a single sink, cluster administrators can get a detailed view of what interactive actions are occuring in real time.

## Warning 

This project is strictly alpha. kubepf injects code that is run in kernel space. Although this code runs on an in-kernel VM and should be safe, there can be perfomance implications - use at your own risk!

## Quickstart 

### Prerequisites

Kernel for eBPF is relatively recent. In order for this project to work, you need to be running at least version 4.8. Additionally, due to a change in the kernel, version 4.13 will not work. However, this change was reversed in 4.14.
