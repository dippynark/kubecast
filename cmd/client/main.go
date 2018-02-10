package main

import (
	"fmt"
	"os"

	"github.com/dippynark/kubepf/pkg/kubepf"
)

func main() {

	err := kubepf.New()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

}
