package kubepf

import (
	"bytes"
	"unsafe"
	"errors"

	bpflib "github.com/iovisor/gobpf/elf"
)

/*
#include <linux/bpf.h>
#include "../../bpf/bpf_tty.h"
*/
import "C"

const (
	bufferSize = 256
)

type TtyWrite struct {
	Count     uint32
	Buffer    [bufferSize]byte
	Timestamp uint64
	Inode uint64
}

func New(channel chan []byte, lostChannel chan uint64) error {

	buf, err := Asset("bpf_tty.o")
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buf)

	m := bpflib.NewModuleFromReader(reader)
	if m == nil {
		return errors.New("error creating new module")
	}

	sectionParams := make(map[string]bpflib.SectionParams)
	sectionParams["maps/tty_writes"] = bpflib.SectionParams{PerfRingBufferPageCount: bufferSize}
	err = m.Load(sectionParams)
	if err != nil {
		return err
	}

	// enable kprobes/kretprobes.
	// For kretprobes, you can configure the maximum number of instances
	// of the function that can be probed simultaneously with maxactive.
	// Here the default value is used by setting maxactive to 0.
	// For kprobes, maxactive is ignored.
	err = m.EnableKprobes(0)
	if err != nil {
		return err
	}

	perfMap, err := bpflib.InitPerfMap(m, "tty_writes", channel, lostChannel)
	if err != nil {
		return err
	}

	perfMap.SetTimestampFunc(ttyWriteTimestamp)

	perfMap.PollStart()

	return nil
}

func ttyWriteTimestamp(data *[]byte) uint64 {
	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))
	return uint64(ttyWrite.timestamp)
}

func TtyWriteToGo(data *[]byte) (ret TtyWrite) {

	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))

	ret.Count = uint32(ttyWrite.count)
	ret.Buffer = *(*[C.BUFSIZE]byte)(unsafe.Pointer(&ttyWrite.buf))
	ret.Timestamp = uint64(ttyWrite.timestamp)
	ret.Inode = uint64(ttyWrite.ino)

	return
}