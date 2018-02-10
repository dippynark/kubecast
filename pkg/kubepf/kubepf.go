package kubepf

import (
	"bytes"
	"fmt"

	bpflib "github.com/iovisor/gobpf/elf"
)

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
	//sectionParams["maps/tty_writes"] = bpflib.SectionParams{PerfRingBufferPageCount: 256}
	err = m.Load(sectionParams)
	if err != nil {
		return err
	}

	return nil
}
