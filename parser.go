package flagforge

import (
	"fmt"
	"io"

	"github.com/mitchellh/mapstructure"
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
	Delimiter string      `mapstructure:"delimiter"`
	Default   interface{} `mapstructure:"default"`
	ShortHelp string      `mapstructure:"short_help"`
	LongHelp  string      `mapstructure:"long_help"`

	// Section groups the flag with others in the generated documentation. It is
	// ignored by the Go generator.
	Section string `mapstructure:"section"`

	// Hide excludes the flag from flag registration and documentation, but keeps
	// its field in the generated configuration struct.
	Hide bool `mapstructure:"hide"`
}

type ParsedConfig struct {
	GoConfig  GoConfig   `mapstructure:"go"`
	Arguments []Argument `mapstructure:"arguments"`
	Flags     []Flag     `mapstructure:"flags"`
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
		return nil, fmt.Errorf("failed to read TOML file at %s: %w", path, err)
	}
	return parseConfig(v)
}

func (p *Parser) ParseReader(r io.Reader) (*ParsedConfig, error) {
	v := getViper()
	if err := v.ReadConfig(r); err != nil {
		return nil, fmt.Errorf("failed to read TOML from reader: %w", err)
	}
	return parseConfig(v)
}

func parseConfig(v *viper.Viper) (*ParsedConfig, error) {
	cfg := ParsedConfig{GoConfig: GoConfig{
		Package:           "pkg",
		ConfigTypeName:    "Config",
		FlagSetName:       "name",
		FlagErrorHandling: "ExitOnError",
	}}
	if err := v.UnmarshalExact(&cfg, func(c *mapstructure.DecoderConfig) {
		c.WeaklyTypedInput = false
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

func getViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("toml")
	return v
}
