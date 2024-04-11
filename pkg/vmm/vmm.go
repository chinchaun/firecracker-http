package vmm

import (
	"context"
	"fmt"
	"open-fire/configs"
	"open-fire/pkg/vmm/chroot"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/hashicorp/go-hclog"
	"github.com/sirupsen/logrus"
)

// StoppedOK is the VMM stopped status.
type StoppedOK = bool

var (
	// StoppedGracefully indicates the machine was stopped gracefully.
	StoppedGracefully = StoppedOK(true)
	// StoppedForcefully indicates that the machine did not stop gracefully
	// and the shutdown had to be forced.
	StoppedForcefully = StoppedOK(false)
)

// Provider abstracts the configuration required to start a VMM.
type Provider interface {
	// Start starts the VMM.
	Start(context.Context) (StartedMachine, error)

	WithHandlersAdapter(firecracker.HandlersAdapter) Provider
}

type defaultProvider struct {
	cniConfig       *configs.CNIConfig
	jailingFcConfig *configs.JailingFirecrackerConfig
	machineConfig   *configs.MachineConfig

	handlersAdapter firecracker.HandlersAdapter
	logger          hclog.Logger
}

// NewDefaultProvider creates a default provider.
func NewDefaultProvider(cniConfig *configs.CNIConfig, jailingFcConfig *configs.JailingFirecrackerConfig, machineConfig *configs.MachineConfig) Provider {
	return &defaultProvider{
		cniConfig:       cniConfig,
		jailingFcConfig: jailingFcConfig,
		machineConfig:   machineConfig,

		handlersAdapter: configs.DefaultFirectackerStrategy(machineConfig),
		logger:          hclog.Default(),
	}
}

func (p *defaultProvider) Start(ctx context.Context) (StartedMachine, error) {

	machineChroot := chroot.NewWithLocation(chroot.LocationFromComponents(p.jailingFcConfig.JailerChrootDirectory(),
		// TODO: MMM este cambio acá con el read link puede romper todo
		p.jailingFcConfig.BinaryFirecracker(),
		p.jailingFcConfig.VMMID()))

	vmmLoggerEntry := logrus.NewEntry(logrus.New())
	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(vmmLoggerEntry),
	}

	if p.machineConfig.LogFcHTTPCalls {
		machineOpts = append(machineOpts, firecracker.
			WithClient(firecracker.NewClient(machineChroot.SocketPath(), vmmLoggerEntry, true)))
	}

	fcConfig, err := configs.NewFcConfigProvider(p.jailingFcConfig, p.machineConfig).
		WithHandlersAdapter(p.handlersAdapter).
		ToSDKConfig()

	if err != nil {
		return &defaultStartedMachine{}, err
	}

	m, err := firecracker.NewMachine(ctx, fcConfig, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating machine: %s", err)
	}
	if err := m.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start machine: %v", err)
	}

	return &defaultStartedMachine{
		cniConfig:       p.cniConfig,
		jailingFcConfig: p.jailingFcConfig,
		machineConfig:   p.machineConfig,
		logger:          p.logger,
		machine:         m,
	}, nil
}

func (p *defaultProvider) WithHandlersAdapter(input firecracker.HandlersAdapter) Provider {
	p.handlersAdapter = input
	return p
}
