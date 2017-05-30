package lib

import "os"

// Config stores parse/report config including tracepoint, time section and so on.
type Config struct {
}

// NewConfig creates a config instance, parsing the config file f.
func NewConfig(f *os.File) *Config {
	return new(Config)
}

// DefaultConfig creates a default config instance.
func DefaultConfig(f *os.File) *Config {
	var cfg = new(Config)
	return cfg
}
