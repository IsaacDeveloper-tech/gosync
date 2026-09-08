package main

import (
	"fmt"
	"os"
)

func main() {
	if err := runWatchCommand(os.Args[1:], WatchCommandOptions{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
