package configs

import (
	"fmt"
	"open-fire/dtos/requests"
	"time"
)

// KillCommandConfig is the kill command configuration.
type KillConfig struct {
	ShutdownTimeout time.Duration `json:"ShutdownTimeout" description:"If the VMM is running and shutdown is called, how long to wait for clean shutdown"`
	VMMID           string
	PID             int `json:"PID"`
	Arch            string
}

// NewKillCommandConfig returns new command configuration.
func NewKillConfig() *KillConfig {
	return &KillConfig{
		ShutdownTimeout: time.Second * 1,
		VMMID:           "",
		PID:             0,
	}
}

// Validate validates the correctness of the configuration.
func (c *KillConfig) Validate() error {
	if c.VMMID == "" {
		return fmt.Errorf("--vmm-id can't be empty")
	}

	if c.Arch == "" {
		return fmt.Errorf("arch can't be empty, values: aarch64, x86_64")
	}

	return nil
}

func (c *KillConfig) WithStopVMRequest(stopRequest *requests.StopVMRequest) error {
	c.VMMID = stopRequest.VMMiD
	c.PID = stopRequest.PID
	c.Arch = stopRequest.Arch
	return c.Validate()

}
