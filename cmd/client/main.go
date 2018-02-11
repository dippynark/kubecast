package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dippynark/kubepf/pkg/kubepf"
)

func main() {

	err := kubepf.New(&kubepf.TtyWriteTracer{})
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	time.Sleep(20 * time.Second)

}
