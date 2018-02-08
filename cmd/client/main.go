package main

/*
#include <stdio.h>
#include <stdlib.h>

void print(char* s) {
 printf("%s\n", s);
}
*/
import "C"

import (
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

  //for i := 0; i < 6; i++ {
	//b, err := ioutil.ReadFile(fmt.Sprintf("bpf/test%d.o", i))
	b, err := ioutil.ReadFile("bpf/bpf_tty.o")
	if err != nil {
		fmt.Print(err)
	}

	err = loadProgram(BPF_PROG_TYPE_KPROBE, unsafe.Pointer(&b), len(b))
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	//}

}

func loadProgram(progType int, insns unsafe.Pointer, insnCnt int) error {

	licenseBuf := "GPL"
	licenseStr := C.CString(licenseBuf)
	defer C.free(unsafe.Pointer(licenseStr))

	logStr := C.CString("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	defer C.free(unsafe.Pointer(logStr))

	lba := struct {
		progType uint32
		//pad0        [4]byte
		insnCnt uint32
		//pad1        [4]byte
		insns    uint64
		license  uint64
		logLevel uint32
		//pad2        [4]byte
		logSize uint32
		//pad3    [4]byte
		logBuf  uint64
		kernVersion uint32
		//pad4        [4]byte
	}{
		progType: uint32(progType),		
		insnCnt:  uint32(insnCnt),
		insns:    uint64(uintptr(insns)),
		license:  uint64(uintptr(unsafe.Pointer(licenseStr))),		
		logLevel: uint32(1),
		logSize:  uint32(50),
		logBuf:   uint64(uintptr(unsafe.Pointer(logStr))),
		//logBuf: uint64(uintptr(unsafe.Pointer(bufStr))),
		// /usr/src/linux-headers-4.13.0-32-generic/include/generated/uapi/linux/version.h
		kernVersion: uint32(265485),
	}

	ret, _, err := unix.Syscall(
		unix.SYS_BPF,
		bpf.BPF_PROG_LOAD,
		uintptr(unsafe.Pointer(&lba)),
		unsafe.Sizeof(lba),
	)

	//fmt.Printf("%s\n", logBuf)
	//cs := C.CString("XXXXXXXXXX")
	C.print(logStr)
	//fmt.Printf("%c\n", *logStr)

	if ret != 0 || err != 0 {
		//fmt.Printf("%#v %d\n", logBuf, unsafe.Sizeof(lba))
		return fmt.Errorf("Unable to load program: ret: %d: %s", int(ret), err)
	}

	return nil
}
