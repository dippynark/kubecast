package main

import (
	"fmt"
	"io/ioutil"
)

func main() {

	_, err := ioutil.ReadFile("bpf/bpf_tty.o")
	if err != nil {
		fmt.Print(err)
	}

}
