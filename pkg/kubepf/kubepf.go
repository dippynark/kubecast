package kubepf

import (
	"bytes"
	"fmt"
	"unsafe"
	"os"

	bpflib "github.com/iovisor/gobpf/elf"
)

/*
#include "../../bpf/bpf_tty.h"
*/
import "C"

const (
	// maxActive configures the maximum number of instances of the probed functions
	// that can be handled simultaneously.
	// This value should be enough to handle typical workloads (for example, some
	// amount of processes blocked on the tty_write syscall).
	maxActive  = 128
	bufferSize = 256
)

type TtyWriteTracer struct {
	lastTimestamp uint64
}

func (t *TtyWriteTracer) Print(ttyWrite TtyWrite) {
	fmt.Printf("%s", ttyWrite.Buffer[0:ttyWrite.Count])
	//fmt.Printf("%d\n", ttyWrite.SessionID)

	if t.lastTimestamp > ttyWrite.Timestamp {
		fmt.Printf("ERROR: late event!\n")
		os.Exit(1)
	}

	t.lastTimestamp = ttyWrite.Timestamp
}

func New(cb Callback) error {

	buf, err := Asset("bpf_tty.o")
	if err != nil {
		return fmt.Errorf("could not find asset: %s", err)
	}
	reader := bytes.NewReader(buf)

	m := bpflib.NewModuleFromReader(reader)
	if m == nil {
		return fmt.Errorf("BPF not supported")
	}

	sectionParams := make(map[string]bpflib.SectionParams)
	sectionParams["maps/tty_writes"] = bpflib.SectionParams{PerfRingBufferPageCount: 256}
	err = m.Load(sectionParams)
	if err != nil {
		return fmt.Errorf("failed to load BPF module: %s", err)
	}

	err = m.EnableKprobes(0)
	if err != nil {
		return fmt.Errorf("failed to enable kprobes: %s", err)
	}

	channel := make(chan []byte)
	lostChannel := make(chan uint64)

	perfMap, err := bpflib.InitPerfMap(m, "tty_writes", channel, lostChannel)
	if err != nil {
		return fmt.Errorf("error initializing perf map: %s", err)
	}

	perfMap.SetTimestampFunc(ttyWriteTimestamp)

	stopChan := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopChan:
				// On stop, stopChan will be closed but the other channels will
				// also be closed shortly after. The select{} has no priorities,
				// therefore, the "ok" value must be checked below.
				return
			case ttyWrite, ok := <-channel:
				if !ok {
					return // see explanation above
				}
				ttyWriteGo := ttyWriteToGo(&ttyWrite)
				//fmt.Printf("%d\n", ttyWrite.Count)
				//fmt.Printf("%s", ttyWriteGo.Buffer[0:ttyWriteGo.Count])
				cb.Print(ttyWriteGo)
			case _, ok := <-lostChannel:
				if !ok {
					return // see explanation above
				}
				//fmt.Printf("%#v\n", lost)
			}
		}
	}()

	perfMap.PollStart()

	return nil
}

type TtyWrite struct {
	Count    uint32
	Buffer   string
	SessionID uint32
	Timestamp uint64
}

func ttyWriteToGo(data *[]byte) (ret TtyWrite) {

	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))

	ret.Count = uint32(ttyWrite.count)
	ret.Buffer = C.GoString(&ttyWrite.buf[0])
	ret.SessionID = uint32(ttyWrite.sessionid)
	ret.Timestamp = uint64(ttyWrite.timestamp)

	return
}

func ttyWriteTimestamp(data *[]byte) uint64 {
	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))
	return uint64(ttyWrite.timestamp)
}