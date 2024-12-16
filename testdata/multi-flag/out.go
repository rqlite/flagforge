// Code generated by go generate; DO NOT EDIT.
package pkg

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// StringSlice wraps a string slice and implements the flag.Value interface.
type StringSliceValue struct {
	ss *[]string
}

func NewStringSliceValue(ss *[]string) *StringSliceValue {
	return &StringSliceValue{ss}
}

// String returns a string representation of the StringSliceValue.
func (s *StringSliceValue) String() string {
	return fmt.Sprintf("%v", *s.ss)
}

// Set sets the value of the StringSliceValue.
func (s *StringSliceValue) Set(value string) error {
	*s.ss = strings.Split(value, ",")
	return nil
}

// Config represents all configuration options.
type Config struct {
	// Node ID
	NodeID string
	// HTTP API bind address
	HTTPAddr string
	// An interval of time
	Interval time.Duration
	// A slice of strings
	List []string
}

// Forge sets up and parses command-line flags.
func Forge(arguments []string) (*flag.FlagSet, *Config, error) {
	config := &Config{}
	fs := flag.NewFlagSet("name", flag.ExitOnError)
	fs.StringVar(&config.NodeID, "-node-id", "", "Node ID")
	fs.StringVar(&config.HTTPAddr, "-http-addr", "localhost:4001", "HTTP API bind address")
	fs.DurationVar(&config.Interval, "-interval", mustParseDuration("10s"), "An interval of time")
	fs.Var(NewStringSliceValue(&config.List), "-list", "A slice of strings")
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
