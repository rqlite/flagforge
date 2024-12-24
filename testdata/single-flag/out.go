// Code generated by go generate; DO NOT EDIT.
package pkg

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config represents all configuration options.
type Config struct {
	// Node ID
	NodeID string
}

// Forge sets up and parses command-line flags.
func Forge(arguments []string) (*flag.FlagSet, *Config, error) {
	config := &Config{}
	fs := flag.NewFlagSet("name", flag.ExitOnError)
	fs.StringVar(&config.NodeID, "-node-id", "", "Node ID")
	if err := fs.Parse(arguments); err != nil {
		return nil, nil, err
	}
	return fs, config, nil
}

func mustParseDuration(d string) time.Duration {
	td, err := time.ParseDuration(d)
	if err != nil {
		panic(err)
	}
	return td
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}

func fmtError(msg string) error {
	return fmt.Errorf(msg)
}

func usage(msg string) {
	fmt.Fprintf(os.Stderr, msg)
}
