package configs

// import (
// 	"github.com/spf13/pflag"
// )

// TracingConfig is the tracing configuration.
type TracingConfig struct {
	// flagBase

	ApplicationName string
	Enable          bool
	HostPort        string
	LogEnable       bool
}

// NewTracingConfig returns a new instance of the configuration.
func NewTracingConfig(appName string) *TracingConfig {
	return &TracingConfig{
		ApplicationName: appName,
	}
}
