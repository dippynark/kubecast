package kubepf

import (
	"syscall"
	"bytes"
	"fmt"
	"unsafe"

	bpflib "github.com/iovisor/gobpf/elf"
)

/*
#include <linux/bpf.h>
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

type sidT struct {
	sid int
}

func New(channel chan []byte, lostChannel chan uint64) error {

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

	// add current session ID to excluded_sids map
	excludedSidsMap := m.Map("excluded_sids")
	sid, _, _ := syscall.Syscall(syscall.SYS_GETSID, 0, 0, 0)
	key := sidT{sid: int(sid)}
	value := 1
	m.UpdateElement(excludedSidsMap, unsafe.Pointer(&key), unsafe.Pointer(&value), C.BPF_ANY)

	perfMap, err := bpflib.InitPerfMap(m, "tty_writes", channel, lostChannel)
	if err != nil {
		return fmt.Errorf("error initializing perf map: %s", err)
	}

	perfMap.SetTimestampFunc(ttyWriteTimestamp)

	perfMap.PollStart()

	return nil
}

func ttyWriteTimestamp(data *[]byte) uint64 {
	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))
	return uint64(ttyWrite.timestamp)
}