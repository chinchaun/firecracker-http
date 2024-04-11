package configs

// CNIConfig provides CNI configuration options.
type CNIConfig struct {
	BinDir   string `json:"BinDir" mapstructure:"BinDir" description:"CNI plugins binaries directory"`
	ConfDir  string `json:"ConfDir" mapstructure:"ConfDir" description:"CNI configuration directory"`
	CacheDir string `json:"CacheDir" mapstructure:"CacheDir" description:"CNI cache directory"`
}

// NewCNIConfig returns a new instance of the configuration.
func NewCNIConfig() *CNIConfig {
	return &CNIConfig{
		BinDir:   "/opt/cni/bin",
		ConfDir:  "/etc/cni/conf.d",
		CacheDir: "/var/lib/cni",
	}
}
