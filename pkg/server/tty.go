package server

const (
	BufferSize   = 256
	HostnameSize = 64
)

type TtyWrite struct {
	Count              uint32
	Buffer             [BufferSize]byte
	Timestamp          uint64
	Inode              uint64
	MountNamespaceInum uint64
	Hostname           [HostnameSize]byte
}
