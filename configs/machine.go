package configs

import (
	"fmt"
	"net"
	"open-fire/dtos/requests"
	"os"
)

// MachineConfig provides machine configuration options.
type MachineConfig struct {
	CNINetworkName    string `json:"CniNetworkName" mapstructure:"CniNetworkName" description:"CNI network within which the build should run; it's recommended to use a dedicated network for build process"`
	CPU               int64  `json:"CPU" mapstructure:"CPU" description:"Number of CPUs for the build VMM"`
	CPUTemplate       string `json:"CPUTemplate" mapstructure:"CPUTemplate" description:"CPU template (empty, C2 or T3)"`
	Smt               bool   `json:"Smt" mapstructure:"Smt" description:"Flag for enabling/disabling simultaneous multithreading. Can be enabled only on x86."`
	IPAddress         string `json:"IPAddress" mapstructure:"IPAddress" description:"IP address to try to allocate to the VM; if not given, a new IP will be allocated"`
	KernelArgs        string `json:"KernelArgs" mapstructure:"KernelArgs" description:"Kernel arguments"`
	Mem               int64  `json:"Mem" mapstructure:"Mem" description:"Amount of memory for the VMM"`
	RootDrivePartUUID string `json:"RootDrivePartuuid" mapstructure:"RootDrivePartuuid" description:"Root drive part UUID"`
	SSHUser           string `json:"SSHUser" mapstructure:"SSHUser" description:"SSH user"`

	LogFcHTTPCalls                 bool            `json:"LogFirecrackerHTTPCalls" mapstructure:"LogFirecrackerHTTPCalls" description:"If set, logs Firecracker HTTP client calls in debug mode"`
	ShutdownGracefulTimeoutSeconds int             `json:"ShutdownGracefulTimeoutSeconds" mapstructure:"ShutdownGracefulTimeoutSeconds" description:"Graceful shutdown timeout before vmm is stopped forcefully"`
	KernelPath                     string          `json:"KernelPath" mapstructure:"KernelPath" description:"The path of the Kernel in the Host Machine"`
	RootFSPath                     string          `json:"RootFSPath" mapstructure:"RootFSPath" description:"The path of the Root File System in the Host Machine"`
	FcAdditionalDrives             []string        `long:"add-drive" description:"Path to additional drive, suffixed with :ro or :rw, can be specified multiple times"`
	FcRootPartUUID                 string          `long:"root-partition" description:"Root partition UUID"`
	FcVsockDevices                 []string        `long:"vsock-device" description:"Vsock interface, specified as PATH:CID. Multiple OK"`
	FcLogFifo                      string          `long:"vmm-log-fifo" description:"FIFO for firecracker logs"`
	FcFifoLogFile                  string          `long:"firecracker-log" short:"l" description:"pipes the fifo contents to the specified file"`
	FcMetricsFifo                  string          `long:"metrics-fifo" description:"FIFO for firecracker metrics"`
	FcMetadata                     *MetadataConfig `json:"Metadata" description:"Metadata validated to be used in the call of SetMetadata in FC"`
	Debug                          bool            `json:"Debug" mapstructure:"Debug" description:"If debug should be enabled"`
	LogLevel                       string          `json:"LogLevel" mapstructure:"LogLevel" description:"LogLevel defines the verbosity of Firecracker logging.  Valid values are Error, Warning, Info (default), and Debug, and are case-sensitive."`

	closers []func() error

	daemonize bool
}

// NewMachineConfig returns a new instance of the configuration.
func NewMachineConfig() *MachineConfig {
	return &MachineConfig{
		CNINetworkName:                 "",
		CPU:                            0,
		CPUTemplate:                    "",
		Smt:                            false,
		IPAddress:                      "",
		KernelArgs:                     "noapic reboot=k panic=1 pci=off nomodules nosmt=force l1tf=full,force 8250.nr_uarts=0 quiet loglevel=1 rw",
		Mem:                            0,
		RootDrivePartUUID:              "",
		SSHUser:                        "",
		KernelPath:                     "",
		RootFSPath:                     "",
		LogFcHTTPCalls:                 false,
		ShutdownGracefulTimeoutSeconds: 30,
		Debug:                          false,
		LogLevel:                       "Info",
		FcMetadata:                     NewMetadataConfig(),
	}
}

// Daemonize returns the configured daemonize setting.
func (c *MachineConfig) Daemonize() bool {
	return c.daemonize
}

// WithDaemonize sets the daemonize setting.
func (c *MachineConfig) WithDaemonize(input bool) *MachineConfig {
	c.daemonize = input
	return c
}

// Validate validates the correctness of the configuration.
func (c *MachineConfig) Validate() error {
	if c.IPAddress != "" {
		if parsedIP := net.ParseIP(c.IPAddress); parsedIP == nil {
			return fmt.Errorf("value of --ip-address is not an IP address")
		}
	}

	if c.KernelPath == "" {
		return fmt.Errorf("kernel path cannot be empty")
	}

	if c.RootFSPath == "" {
		return fmt.Errorf("rootfs path cannot be empty")
	}

	logLevel := []string{"Error", "Warning", "Info", "Debug"}

	if !containsString(logLevel, c.LogLevel) {
		return fmt.Errorf("the log level is invalid")
	}

	if c.CPU < 1 {
		return fmt.Errorf("number of VcpuCount cannot be lower than 1")
	}

	if c.Mem < 128 {
		return fmt.Errorf("number of MemSizeMib cannot be lower than 128")
	}

	return nil
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func (c *MachineConfig) WithCreateVMRequest(createVM *requests.CreateVMRequest) error {

	if os.Getenv("ENV") == "PROD" {
		c.LogLevel = "Error"
	}

	c.KernelPath = createVM.KernelPath
	c.RootFSPath = createVM.RootDrivePath
	c.CNINetworkName = createVM.CniNetworkName

	if createVM.AdditionalDrives != "" {
		c.FcAdditionalDrives = append(c.FcAdditionalDrives, createVM.AdditionalDrives)
	}

	if createVM.Metadata.Data != "" {
		c.FcMetadata = (*MetadataConfig)(&createVM.Metadata)
	}

	c.Debug = createVM.Debug
	c.CPU = createVM.VcpuCount
	c.Mem = createVM.MemSizeMib
	c.Smt = createVM.EnableSmt

	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}
