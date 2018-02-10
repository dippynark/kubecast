package kubepf

import (
	"bytes"
	"fmt"

	bpflib "github.com/iovisor/gobpf/elf"
)

// maxActive configures the maximum number of instances of the probed functions
// that can be handled simultaneously.
// This value should be enough to handle typical workloads (for example, some
// amount of processes blocked on the tty_write syscall).
const maxActive = 128

func New() error {

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

	//perfMap.SetTimestampFunc(ttyWriteTimestamp)

	stopChan := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopChan:
				// On stop, stopChan will be closed but the other channels will
				// also be closed shortly after. The select{} has no priorities,
				// therefore, the "ok" value must be checked below.
				return
			case data, ok := <-channel:
				if !ok {
					return // see explanation above
				}
				fmt.Printf("%#v\n", data)
			case lost, ok := <-lostChannel:
				if !ok {
					return // see explanation above
				}
				fmt.Printf("%#v\n", lost)
			}
		}
	}()

	perfMap.PollStart()

	return nil
}
