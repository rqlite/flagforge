package flagforge

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
)

// GoConfig represents the configuration for the generated Go code.
type GoConfig struct {
	Package           string `mapstructure:"package"`
	ConfigTypeName    string `mapstructure:"config_type_name"`
	FlagSetUsage      string `mapstructure:"flag_set_usage"`
	FlagSetName       string `mapstructure:"flag_set_name"`
	FlagErrorHandling string `mapstructure:"flag_error_handling"`
}

// Argument represents a single argument configuration.
type Argument struct {
	Name      string `mapstructure:"name"`
	Type      string `mapstructure:"type"`
	Required  bool   `mapstructure:"required"`
	ShortHelp string `mapstructure:"short_help"`
	LongHelp  string `mapstructure:"long_help"`
}

// Flag represents a single flag configuration.
type Flag struct {
	Name      string      `mapstructure:"name"`
	CLI       string      `mapstructure:"cli"`
	Type      string      `mapstructure:"type"`
	Default   interface{} `mapstructure:"default"`
	ShortHelp string      `mapstructure:"short_help"`
	LongHelp  string      `mapstructure:"long_help"`
}

type ParsedConfig struct {
	GoConfig  GoConfig
	Arguments []Argument
	Flags     []Flag
}

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParsePath(path string) (*ParsedConfig, error) {
	v := getViper()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read TOML file: %w", err)
	}
	return parseConfig(v)
}

func (p *Parser) ParseReader(r io.Reader) (*ParsedConfig, error) {
	v := getViper()
	if err := viper.ReadConfig(r); err != nil {
		return nil, fmt.Errorf("failed to read TOML from reader: %w", err)
	}
	return parseConfig(v)
}

func parseConfig(v *viper.Viper) (*ParsedConfig, error) {
	goConfig := GoConfig{
		Package:           "pkg",
		ConfigTypeName:    "Config",
		FlagSetName:       "name",
		FlagErrorHandling: "ExitOnError",
	}

	if err := v.UnmarshalKey("go", &goConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal go config: %w", err)
	}

	var args []Argument
	if err := v.UnmarshalKey("arguments", &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	var flags []Flag
	if err := v.UnmarshalKey("flags", &flags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}
	return &ParsedConfig{
		GoConfig:  goConfig,
		Arguments: args,
		Flags:     flags,
	}, nil
}

func getViper() *viper.Viper {
	return viper.New()
}
