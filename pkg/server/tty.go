package server

const (
	BufferSize        = 256
	HostnameSize      = 64
	PodNameSize       = 253
	ContainerNameSize = 253
	PodNamespaceSize  = 253
	PodUIDSize        = 32
)

type TtyWrite struct {
	Count              uint32
	Buffer             [BufferSize]byte
	Timestamp          uint64
	Inode              uint64
	MountNamespaceInum uint64
	Hostname           [HostnameSize]byte
	ContainerName      [ContainerNameSize]byte
	PodName            [PodNameSize]byte
	PodNamespace       [PodNamespaceSize]byte
	PodUID             [PodUIDSize]byte
}
