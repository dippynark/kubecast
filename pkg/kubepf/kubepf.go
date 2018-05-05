package kubepf

import (
	"syscall"
	"bytes"
	"fmt"
	"unsafe"
	"net/http"
	"errors"

	bpflib "github.com/iovisor/gobpf/elf"
	"github.com/golang/glog"
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
	sessionIDHTTPHeader  = "X-Session-ID"
)

type TtyWriteTracer struct {
	lastTimestamp uint64
	module *bpflib.Module
}

type ttyWrite struct {
	Count     uint32
	Buffer    string
	Timestamp uint64
}

type sidT struct {
	sid int
}

func New(channel chan []byte, lostChannel chan uint64) (*TtyWriteTracer, error) {

	buf, err := Asset("bpf_tty.o")
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(buf)

	m := bpflib.NewModuleFromReader(reader)
	if m == nil {
		return nil, errors.New("error creating new module")
	}

	sectionParams := make(map[string]bpflib.SectionParams)
	sectionParams["maps/tty_writes"] = bpflib.SectionParams{PerfRingBufferPageCount: 256}
	err = m.Load(sectionParams)
	if err != nil {
		return nil, err
	}

	err = m.EnableKprobes(0)
	if err != nil {
		return nil, err
	}

	// add current session ID to excluded_sids map
	excludedSidsMap := m.Map("excluded_sids")
	sid, _, _ := syscall.Syscall(syscall.SYS_GETSID, 0, 0, 0)
	key := sidT{sid: int(sid)}
	value := 1
	m.UpdateElement(excludedSidsMap, unsafe.Pointer(&key), unsafe.Pointer(&value), C.BPF_ANY)

	perfMap, err := bpflib.InitPerfMap(m, "tty_writes", channel, lostChannel)
	if err != nil {
		return nil, err
	}

	perfMap.SetTimestampFunc(ttyWriteTimestamp)

	perfMap.PollStart()

	return &TtyWriteTracer{module: m}, nil
}

func ttyWriteTimestamp(data *[]byte) uint64 {
	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))
	return uint64(ttyWrite.timestamp)
}

func (t *TtyWriteTracer) SetActiveSID(sid int) error {
	activeSidMap := t.module.Map("active_sid")
	key := 0
	value := sidT{sid: int(sid)}
	return t.module.UpdateElement(activeSidMap, unsafe.Pointer(&key), unsafe.Pointer(&value), C.BPF_ANY)
}

func TtyWriteToGo(data *[]byte) (ret ttyWrite) {

	ttyWrite := (*C.struct_tty_write_t)(unsafe.Pointer(&(*data)[0]))

	ret.Count = uint32(ttyWrite.count)
	ret.Buffer = C.GoString(&ttyWrite.buf[0])
	ret.Timestamp = uint64(ttyWrite.timestamp)

	return
}

func (t *TtyWriteTracer) Upload(ttyWrite ttyWrite, server string) {

	payload := bytes.NewBufferString(ttyWrite.Buffer[0:ttyWrite.Count])
	req, err := http.NewRequest("POST", server, payload)
	if err != nil {
		glog.Errorf("error creating new request: %s", err)
		return
	}
	req.Header.Set(sessionIDHTTPHeader, fmt.Sprintf("%d", ttyWrite.SessionID))
	req.Header.Set("Content-Type", "binary/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("error uploading data: %s", err)
		return
	}
	glog.Infof("uploaded %d bytes", ttyWrite.Count)
	defer resp.Body.Close()

	if t.lastTimestamp > ttyWrite.Timestamp {
		glog.Fatal("late event!")
	}

	t.lastTimestamp = ttyWrite.Timestamp
}
