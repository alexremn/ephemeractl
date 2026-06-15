package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run())
}

// run is the real entrypoint; main only translates its result to an exit code.
func run() int {
	fmt.Println("ephemeractl: not yet implemented")
	return 0
}
