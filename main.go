package main

import (
	"fmt"
	"os"

	gosync "gosync/src"
)

func main() {
	arguments := os.Args[1:]
	var err error
	if len(arguments) > 0 && arguments[0] == "configure" {
		err = gosync.RunConfigureCommand(arguments, gosync.ConfigureCommandOptions{})
	} else {
		err = gosync.RunWatchCommand(arguments, gosync.WatchCommandOptions{})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
