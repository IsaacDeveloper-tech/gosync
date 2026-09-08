package main

import (
	"fmt"
	"os"

	gosync "gosync/src"
)

func main() {
	if err := gosync.RunWatchCommand(os.Args[1:], gosync.WatchCommandOptions{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
