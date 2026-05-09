// Command kafko is a modern, container-friendly Kafka CLI.
package main

import (
	"fmt"
	"os"

	"github.com/darioajr/kafko/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "kafko: %v\n", err)
		os.Exit(1)
	}
}
