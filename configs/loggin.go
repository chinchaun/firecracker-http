package configs

import (
	"os"

	"github.com/hashicorp/go-hclog"
)

// LogConfig represents logging configuration.
type LogConfig struct {
	LogLevel  string
	LogAsJSON bool
}

// NewLogginConfig returns a new logging configuration.
func NewLogginConfig() *LogConfig {
	return &LogConfig{
		LogLevel:  "info",
		LogAsJSON: false,
	}
}

type LoggerOpts struct {
	// optional arguments
	Name      string
	LogLevel  string
	LogAsJSON bool
}

// NewLogger returns a new configured logger.
func (c *LogConfig) NewLogger(opts LoggerOpts) hclog.Logger {
	env := os.Getenv("ENV")
	if env == "PROD" {
		c.LogLevel = "error"
	} else {
		c.LogLevel = "debug"
	}

	if opts.LogLevel != "" {
		c.LogLevel = opts.LogLevel
	}

	if opts.LogAsJSON {
		c.LogAsJSON = opts.LogAsJSON
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       opts.Name,
		Level:      hclog.LevelFromString(c.LogLevel),
		Color:      hclog.AutoColor,
		JSONFormat: c.LogAsJSON,
	})
}
