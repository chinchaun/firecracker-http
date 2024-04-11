package configs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"open-fire/pkg/strategy/arbitrary"
	"open-fire/utils"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/opentracing/opentracing-go/log"
)

const (
	rwDeviceSuffix = ":rw"
	roDeviceSuffix = ":ro"
)

var (

	// error parsing blockdevices
	errInvalidDriveSpecificationNoSuffix = errors.New("invalid drive specification. Must have :rw or :ro suffix")
	errInvalidDriveSpecificationNoPath   = errors.New("invalid drive specification. Must have path")

	// error parsing vsock
	errUnableToParseVsockDevices = errors.New("unable to parse vsock devices")
	errUnableToParseVsockCID     = errors.New("unable to parse vsock CID as a number")

	errConflictingLogOpts        = errors.New("vmm-log-fifo and firecracker-log cannot be used together")
	errUnableToCreateFifoLogFile = errors.New("failed to create fifo log file")
)

// DefaultVethIfaceName is the default veth interface name.
const DefaultVethIfaceName = "veth"

// DefaultFirectackerStrategy returns an instance of the default Firecracker Jailer strategy for a given machine config.
func DefaultFirectackerStrategy(machineConfig *MachineConfig) arbitrary.PlacingStrategy {
	return arbitrary.NewStrategy(func() *arbitrary.HandlerPlacement {
		return arbitrary.NewHandlerPlacement(firecracker.
			LinkFilesHandler(filepath.Base(machineConfig.KernelPath)),
			firecracker.CreateLogFilesHandlerName)
	})
}

// FcConfigProvider is a Firecracker SDK configuration builder provider.
type FcConfigProvider interface {
	ToSDKConfig() (firecracker.Config, error)
	WithHandlersAdapter(firecracker.HandlersAdapter) FcConfigProvider
}

type defaultFcConfigProvider struct {
	jailingFcConfig *JailingFirecrackerConfig
	machineConfig   *MachineConfig

	fcStrategy firecracker.HandlersAdapter
}

// NewFcConfigProvider creates a new builder provider.
func NewFcConfigProvider(jailingFcConfig *JailingFirecrackerConfig, machineConfig *MachineConfig) FcConfigProvider {
	return &defaultFcConfigProvider{
		jailingFcConfig: jailingFcConfig,
		machineConfig:   machineConfig,
	}
}

func (c *defaultFcConfigProvider) ToSDKConfig() (firecracker.Config, error) {

	if c.machineConfig.Debug {
		c.machineConfig.KernelArgs = c.machineConfig.KernelArgs + " console=ttyS0"
	}

	// console stick to terminal debug
	// c.machineConfig.KernelArgs = "console=ttyS0 reboot=k panic=1 pci=off"

	NICs, err := c.getNetwork()
	if err != nil {
		return firecracker.Config{}, err
	}
	// BlockDevices
	blockDevices, err := c.getBlockDevices()
	if err != nil {
		return firecracker.Config{}, err
	}

	// vsocks
	vsocks, err := parseVsocks(c.machineConfig.FcVsockDevices)
	if err != nil {
		return firecracker.Config{}, err
	}

	// c.machineConfig.FcFifoLogFile = "/some-well-known-path/logs/firecracker.log"
	// c.machineConfig.FcMetricsFifo = "/some-well-known-path/logs/metrics"
	// fifos
	fifo, err := c.handleFifos()
	if err != nil {
		return firecracker.Config{}, err
	}

	return firecracker.Config{
		SocketPath:        "", // given via Jailer
		LogFifo:           c.machineConfig.FcLogFifo,
		LogLevel:          c.machineConfig.LogLevel,
		MetricsFifo:       c.machineConfig.FcMetricsFifo,
		FifoLogWriter:     fifo,
		KernelImagePath:   c.machineConfig.KernelPath,
		KernelArgs:        c.machineConfig.KernelArgs,
		NetNS:             c.jailingFcConfig.NetNS,
		Drives:            blockDevices,
		NetworkInterfaces: NICs,
		VsockDevices:      vsocks,
		MmdsVersion:       firecracker.MMDSv2,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:   firecracker.Int64(c.machineConfig.CPU),
			CPUTemplate: models.CPUTemplate(c.machineConfig.CPUTemplate),
			Smt:         firecracker.Bool(c.machineConfig.Smt),
			MemSizeMib:  firecracker.Int64(c.machineConfig.Mem),
		},
		JailerCfg: &firecracker.JailerConfig{
			GID:           firecracker.Int(c.jailingFcConfig.JailerGID),
			UID:           firecracker.Int(c.jailingFcConfig.JailerUID),
			ID:            c.jailingFcConfig.VMMID(),
			NumaNode:      firecracker.Int(c.jailingFcConfig.JailerNumeNode),
			ExecFile:      c.jailingFcConfig.BinaryFirecracker(),
			JailerBinary:  c.jailingFcConfig.BinaryJailer,
			ChrootBaseDir: c.jailingFcConfig.ChrootBase,
			Daemonize:     c.machineConfig.Daemonize(),
			ChrootStrategy: func() firecracker.HandlersAdapter {
				if c.fcStrategy == nil {
					return DefaultFirectackerStrategy(c.machineConfig)
				}
				return c.fcStrategy
			}(),
			Stdout:        os.Stdout,
			Stderr:        os.Stderr,
			Stdin:         os.Stdin,
			CgroupVersion: "2",
		},
		VMID: c.jailingFcConfig.VMMID(),
	}, nil
}

func (c *defaultFcConfigProvider) WithHandlersAdapter(input firecracker.HandlersAdapter) FcConfigProvider {
	c.fcStrategy = input
	return c
}

func (c *defaultFcConfigProvider) getNetwork() ([]firecracker.NetworkInterface, error) {

	if c.machineConfig.CNINetworkName == "" {
		return nil, fmt.Errorf("cni network of machine cannot be empty")
	}

	var NICs []firecracker.NetworkInterface

	nic := firecracker.NetworkInterface{
		CNIConfiguration: &firecracker.CNIConfiguration{
			NetworkName: c.machineConfig.CNINetworkName,
			IfName:      DefaultVethIfaceName + utils.RandStringBytes(11),
			Args: func() [][2]string {
				return [][2]string{}
			}(),
		},
		AllowMMDS: true,
		InRateLimiter: &models.RateLimiter{
			Bandwidth: &models.TokenBucket{
				Size:       firecracker.Int64(0),
				RefillTime: firecracker.Int64(0),
			},
			Ops: &models.TokenBucket{
				Size:       firecracker.Int64(0),
				RefillTime: firecracker.Int64(0),
			},
		},
		OutRateLimiter: &models.RateLimiter{
			Bandwidth: &models.TokenBucket{
				Size:       firecracker.Int64(0),
				RefillTime: firecracker.Int64(0),
			},
			Ops: &models.TokenBucket{
				Size:       firecracker.Int64(0),
				RefillTime: firecracker.Int64(0),
			},
		},
	}
	NICs = append(NICs, nic)

	return NICs, nil
}

// constructs a list of drives from the options config
func (c *defaultFcConfigProvider) getBlockDevices() ([]models.Drive, error) {
	blockDevices, err := parseBlockDevices(c.machineConfig.FcAdditionalDrives)
	if err != nil {
		return nil, err
	}

	rootDrivePath, readOnly := parseDevice(c.machineConfig.RootFSPath)
	rootDrive := models.Drive{
		DriveID:      firecracker.String("1"),
		PathOnHost:   firecracker.String(rootDrivePath),
		IsReadOnly:   firecracker.Bool(readOnly),
		IsRootDevice: firecracker.Bool(true),
		Partuuid:     c.machineConfig.FcRootPartUUID,
	}
	blockDevices = append(blockDevices, rootDrive)
	return blockDevices, nil
}

// given a []string in the form of path:suffix converts to []models.Drive
func parseBlockDevices(entries []string) ([]models.Drive, error) {
	devices := []models.Drive{}

	for i, entry := range entries {
		path := ""
		readOnly := true

		if strings.HasSuffix(entry, rwDeviceSuffix) {
			readOnly = false
			path = strings.TrimSuffix(entry, rwDeviceSuffix)
		} else if strings.HasSuffix(entry, roDeviceSuffix) {
			path = strings.TrimSuffix(entry, roDeviceSuffix)
		} else {
			return nil, errInvalidDriveSpecificationNoSuffix
		}

		if path == "" {
			return nil, errInvalidDriveSpecificationNoPath
		}

		if _, err := os.Stat(path); err != nil {
			return nil, err
		}

		e := models.Drive{
			// i + 2 represents the drive ID. We will reserve 1 for root.
			DriveID:      firecracker.String(strconv.Itoa(i + 2)),
			PathOnHost:   firecracker.String(path),
			IsReadOnly:   firecracker.Bool(readOnly),
			IsRootDevice: firecracker.Bool(false),
		}
		devices = append(devices, e)
	}
	return devices, nil
}

// Given a string in the form of path:suffix return the path and read-only marker
func parseDevice(entry string) (path string, readOnly bool) {
	if strings.HasSuffix(entry, roDeviceSuffix) {
		return strings.TrimSuffix(entry, roDeviceSuffix), true
	}

	return strings.TrimSuffix(entry, rwDeviceSuffix), false
}

// Given a list of string representations of vsock devices,
// return a corresponding slice of machine.VsockDevice objects
func parseVsocks(devices []string) ([]firecracker.VsockDevice, error) {
	var result []firecracker.VsockDevice
	for _, entry := range devices {
		fields := strings.Split(entry, ":")
		if len(fields) != 2 || len(fields[0]) == 0 || len(fields[1]) == 0 {
			return []firecracker.VsockDevice{}, errUnableToParseVsockDevices
		}
		CID, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return []firecracker.VsockDevice{}, errUnableToParseVsockCID
		}
		dev := firecracker.VsockDevice{
			Path: fields[0],
			CID:  uint32(CID),
		}
		result = append(result, dev)
	}
	return result, nil
}

// handleFifos will see if any fifos need to be generated and if a fifo log
// file should be created.
func (c *defaultFcConfigProvider) handleFifos() (io.Writer, error) {
	// these booleans are used to check whether or not the fifo queue or metrics
	// fifo queue needs to be generated. If any which need to be generated, then
	// we know we need to create a temporary directory. Otherwise, a temporary
	// directory does not need to be created.
	generateFifoFilename := false
	generateMetricFifoFilename := false
	var err error
	var fifo io.WriteCloser

	if len(c.machineConfig.FcFifoLogFile) > 0 {
		if len(c.machineConfig.FcLogFifo) > 0 {
			return nil, errConflictingLogOpts
		}
		generateFifoFilename = true
		// if a fifo log file was specified via the CLI then we need to check if
		// metric fifo was also specified. If not, we will then generate that fifo
		if len(c.machineConfig.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
		if fifo, err = createFifoFileLogs(c.machineConfig.FcFifoLogFile); err != nil {
			return nil, fmt.Errorf("%s: %v", errUnableToCreateFifoLogFile.Error(), err)
		}
		c.machineConfig.addCloser(func() error {
			return fifo.Close()
		})

	} else if len(c.machineConfig.FcLogFifo) > 0 || len(c.machineConfig.FcMetricsFifo) > 0 {
		// this checks to see if either one of the fifos was set. If at least one
		// has been set we check to see if any of the others were not set. If one
		// isn't set, we will generate the proper file path.
		if len(c.machineConfig.FcLogFifo) == 0 {
			generateFifoFilename = true
		}

		if len(c.machineConfig.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
	}

	if generateFifoFilename || generateMetricFifoFilename {
		dir, err := os.MkdirTemp(os.TempDir(), "fcfifo")
		if err != nil {
			return fifo, fmt.Errorf("fail to create temporary directory: %v", err)
		}
		c.machineConfig.addCloser(func() error {
			return os.RemoveAll(dir)
		})
		if generateFifoFilename {
			c.machineConfig.FcLogFifo = filepath.Join(dir, "fc_fifo")
		}

		if generateMetricFifoFilename {
			c.machineConfig.FcMetricsFifo = filepath.Join(dir, "fc_metrics_fifo")
		}
	}

	return fifo, nil
}

func (opts *MachineConfig) addCloser(c func() error) {
	opts.closers = append(opts.closers, c)
}

func (opts *MachineConfig) Close() {
	for _, closer := range opts.closers {
		err := closer()
		if err != nil {
			log.Error(err)
		}
	}
}

func createFifoFileLogs(fifoPath string) (*os.File, error) {
	return os.OpenFile(fifoPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}
