package main

import (
	"C"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/cilium/cilium/pkg/bpf"
)
import (
	"fmt"
	"io/ioutil"
)

const (
	bufferSize           = 256
	sessionIDHTTPHeader  = "X-Session-ID"
	defaultServerAddress = "localhost"
	defaultPort          = 5050
)

const (
	BPF_PROG_TYPE_UNSPEC        = 0
	BPF_PROG_TYPE_SOCKET_FILTER = 1
	BPF_PROG_TYPE_KPROBE        = 2
	BPF_PROG_TYPE_SCHED_CLS     = 3
	BPF_PROG_TYPE_SCHED_ACT     = 4
)

type ttyWrite struct {
	Count     int32
	Buf       [bufferSize]byte
	SessionID int32
}

func main() {

	b, err := ioutil.ReadFile("bpf/bbf_tty.o")
	if err != nil {
		fmt.Print(err)
	}

	err = loadProgram(BPF_PROG_TYPE_KPROBE, unsafe.Pointer(&b), len(b))
	if err != nil {
		fmt.Printf("%s", err)
		return
	}

}

func loadProgram(progType int, insns unsafe.Pointer, insnCnt int) error {

	//logBuf := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	//bufStr := C.CString(logBuf)
	//defer C.free(unsafe.Pointer(bufStr))

	//fmt.Printf("%d\n", uint32(insnCnt))

	lba := struct {
		progType    uint32
		pad0        [4]byte
		insnCnt     uint32
		pad1        [4]byte
		insns       uint64
		license     uint64
		logLevel    uint32
		pad2        [4]byte
		logSize     uint32
		pad3        [4]byte
		logBuf      uint64
		kernVersion uint32
		pad4        [4]byte
	}{
		progType: uint32(progType),
		insns:    uint64(uintptr(insns)),
		insnCnt:  uint32(insnCnt),
		license:  uint64(uintptr(0)),
		logBuf:   uint64(uintptr(0)),
		//logBuf: uint64(uintptr(unsafe.Pointer(bufStr))),
		logSize:     uint32(0),
		logLevel:    uint32(0),
		kernVersion: uint32(4),
	}

	ret, _, err := unix.Syscall(
		unix.SYS_BPF,
		bpf.BPF_PROG_LOAD,
		uintptr(unsafe.Pointer(&lba)),
		unsafe.Sizeof(lba),
	)

	if ret != 0 || err != 0 {
		//fmt.Printf("%#v %d\n", logBuf, unsafe.Sizeof(lba))
		return fmt.Errorf("Unable to load program: ret: %d: %s", int(ret), err)
	}

	return nil
}
