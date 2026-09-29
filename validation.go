package flagforge

import (
	"fmt"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// validateConfig operates on the generator's private copy of the configuration.
func validateConfig(cfg *ParsedConfig) error {
	if !validIdentifier(cfg.GoConfig.Package) {
		return fmt.Errorf("invalid Go package name %q", cfg.GoConfig.Package)
	}
	name := cfg.GoConfig.ConfigTypeName
	if !validIdentifier(name) {
		return fmt.Errorf("invalid configuration type name %q", name)
	}
	if name == "main" && cfg.GoConfig.Package == "main" {
		return fmt.Errorf("configuration type name %q conflicts with the main entry point", name)
	}
	// These names collide with declarations, imports, or predeclared identifiers
	// used by the generated source.
	switch name {
	case "init", "arguments", "Forge", "mustParseDuration", "splitString", "fmtError", "usage",
		"errors", "flag", "fmt", "os", "strings", "time",
		"string", "bool", "int", "int64", "uint64", "error", "nil", "len", "true", "false", "panic":
		return fmt.Errorf("configuration type name %q conflicts with generated code", name)
	}
	switch cfg.GoConfig.FlagErrorHandling {
	case "ContinueOnError", "ExitOnError", "PanicOnError":
	default:
		return fmt.Errorf("invalid flag error handling %q", cfg.GoConfig.FlagErrorHandling)
	}

	fields := make(map[string]bool)
	addField := func(name string) error {
		if !validIdentifier(name) {
			return fmt.Errorf("invalid field name %q", name)
		}
		if fields[name] {
			return fmt.Errorf("duplicate field name %q", name)
		}
		fields[name] = true
		return nil
	}
	optional := false
	for _, arg := range cfg.Arguments {
		if err := addField(arg.Name); err != nil {
			return err
		}
		if arg.Type != "string" {
			return fmt.Errorf("argument %s has unsupported type %q; only string is supported", arg.Name, arg.Type)
		}
		if arg.Required && optional {
			return fmt.Errorf("required argument %s follows an optional argument", arg.Name)
		}
		optional = optional || !arg.Required
	}
	cliNames := make(map[string]bool)
	for i := range cfg.Flags {
		flag := &cfg.Flags[i]
		if err := addField(flag.Name); err != nil {
			return err
		}
		switch flag.Type {
		case "string", "filepath", "bool", "int", "int64", "uint64", "time.Duration", "[]string":
		default:
			return fmt.Errorf("flag %s has unsupported type %q", flag.Name, flag.Type)
		}
		if flag.Hide {
			// Hidden fields have no registration, so CLI settings and defaults
			// are unused. Do not validate or retain their default values.
			flag.Default = nil
			continue
		}
		if flag.CLI == "" || strings.HasPrefix(flag.CLI, "-") || strings.Contains(flag.CLI, "=") || strings.IndexFunc(flag.CLI, unicode.IsSpace) >= 0 {
			return fmt.Errorf("flag %s has invalid CLI name %q; use a name without leading dashes, whitespace, or '='", flag.Name, flag.CLI)
		}
		if cliNames[flag.CLI] {
			return fmt.Errorf("duplicate CLI name %q", flag.CLI)
		}
		cliNames[flag.CLI] = true
		if err := normalizeDefault(flag); err != nil {
			return fmt.Errorf("flag %s: %w", flag.Name, err)
		}
	}
	return nil
}

func validIdentifier(s string) bool {
	return s != "_" && token.IsIdentifier(s)
}

func normalizeDefault(flag *Flag) error {
	switch flag.Type {
	case "string", "filepath", "[]string", "time.Duration":
		if flag.Default == nil {
			flag.Default = ""
			if flag.Type == "time.Duration" {
				flag.Default = "0s"
			}
		}
		s, ok := flag.Default.(string)
		if !ok {
			return fmt.Errorf("%s default must be a string", flag.Type)
		}
		if flag.Type == "time.Duration" {
			if _, err := time.ParseDuration(s); err != nil {
				return fmt.Errorf("invalid duration default: %w", err)
			}
		}
		if flag.Type == "[]string" && flag.Delimiter == "" {
			flag.Delimiter = ","
		}
	case "bool":
		if flag.Default == nil {
			flag.Default = false
		}
		if _, ok := flag.Default.(bool); !ok {
			return fmt.Errorf("bool default must be a boolean")
		}
	case "int", "int64", "uint64":
		if flag.Default == nil {
			flag.Default = 0
		}
		v := reflect.ValueOf(flag.Default)
		var literal string
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			literal = strconv.FormatInt(v.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			literal = strconv.FormatUint(v.Uint(), 10)
		default:
			return fmt.Errorf("%s default must be an integer", flag.Type)
		}
		var err error
		if flag.Type == "uint64" {
			flag.Default, err = strconv.ParseUint(literal, 10, 64)
		} else {
			bits := 64
			if flag.Type == "int" {
				bits = strconv.IntSize
			}
			flag.Default, err = strconv.ParseInt(literal, 10, bits)
		}
		if err != nil {
			return fmt.Errorf("default %s is out of range for %s", literal, flag.Type)
		}
	}
	return nil
}
