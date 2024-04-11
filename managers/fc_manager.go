package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"open-fire/configs"
	"open-fire/pkg/strategy"
	"open-fire/pkg/strategy/arbitrary"
	"open-fire/pkg/vmm"
	"open-fire/pkg/vmm/pid"
	"open-fire/utils"
	"os"
	"strconv"
	"strings"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/hashicorp/go-hclog"
)

var (
	logConfig     = configs.NewLogginConfig()
	tracingConfig = configs.NewTracingConfig("open-fire-vmm-run")
	cniConfig     = configs.NewCNIConfig()
)

type FireCrackerManager struct {
}

func CreateFCManagerInstance() *FireCrackerManager {
	return &FireCrackerManager{}
}

func (instance *FireCrackerManager) StartVM(machineConfig *configs.MachineConfig, jailingFcConfig *configs.JailingFirecrackerConfig) (*firecracker.Machine, error) {

	cleanup := utils.NewDefers()
	defer cleanup.CallAll()

	rootLogger := logConfig.NewLogger(configs.LoggerOpts{Name: "run"})

	validatingConfigs := []configs.ValidatingConfig{
		jailingFcConfig,
	}

	for _, validatingConfig := range validatingConfigs {
		if err := validatingConfig.Validate(); err != nil {
			errorMsg := fmt.Errorf("configuration is invalid, reason: %s", err)
			rootLogger.Error(errorMsg.Error())
			return nil, errorMsg
		}
	}

	rootLogger.Trace("configuring tracing", "enabled", tracingConfig.Enable, "application-name", tracingConfig.ApplicationName)

	vmmStrategy := configs.DefaultFirectackerStrategy(machineConfig).
		AddRequirements(func() *arbitrary.HandlerPlacement {
			// add this one after the previous one so by he logic,
			// this one will be placed and executed before the first one
			return arbitrary.NewHandlerPlacement(strategy.
				NewMetadataExtractorHandler(rootLogger, machineConfig.FcMetadata), firecracker.CreateBootSourceHandlerName)
		})

	vmmProvider := vmm.NewDefaultProvider(cniConfig, jailingFcConfig, machineConfig).
		WithHandlersAdapter(vmmStrategy)

	vmmCtx, vmmCancel := context.WithCancel(context.Background())

	cleanup.Add(func() {
		vmmCancel()
	})

	cleanup.Trigger(false)

	startedMachine, runErr := vmmProvider.Start(vmmCtx)
	if runErr != nil {
		errorMsg := fmt.Errorf("firecracker VMM did not start, run failed, reason: %s", runErr)
		rootLogger.Error(errorMsg.Error())
		return nil, errorMsg
	}

	return startedMachine.RunningMachine(), nil

}

func (instance *FireCrackerManager) StopVM(killCfg *configs.KillConfig, jailingFcConfig *configs.JailingFirecrackerConfig) (string, error) {
	rootLogger := logConfig.NewLogger(configs.LoggerOpts{
		Name: "kill",
	})

	jailingFcConfig.WithVMMID(killCfg.VMMID)

	var runningPid *pid.RunningVMMPID = nil

	if killCfg.PID != 0 {
		runningPid = &pid.RunningVMMPID{
			Pid: killCfg.PID,
		}
	}

	validatingConfigs := []configs.ValidatingConfig{
		jailingFcConfig,
	}

	for _, validatingConfig := range validatingConfigs {
		if err := validatingConfig.Validate(); err != nil {
			errorMsg := fmt.Errorf("configuration is invalid, reason: %s", err)
			rootLogger.Error(errorMsg.Error())
			return "", errorMsg

		}
	}

	rootLogger.Info(jailingFcConfig.JailerChrootDirectory())

	socketPath, hasSocket, existsErr := jailingFcConfig.SocketPathIfExists()

	if existsErr != nil {
		errorMsg := fmt.Errorf("failed checking if the VMM socket file exists, reason: %s", existsErr)
		rootLogger.Error(errorMsg.Error())
		removeJailerChrootDirectory(rootLogger, *jailingFcConfig)
		return "", errorMsg
	}

	resultAarch64 := ""

	if hasSocket {

		rootLogger.Info("stopping VMM")
		if killCfg.Arch == "x86_64" {
			err := instance.stop_x86_64(socketPath, runningPid, rootLogger)
			if err != nil {
				return "", err
			}
		} else if killCfg.Arch == "aarch64" {
			result, err := instance.stop_aarch64(socketPath, runningPid, rootLogger)
			if err != nil {
				return "", err
			}
			resultAarch64 = result
		} else {
			errorMsg := fmt.Errorf("arch not recognized: %s, please use x86_64 or aarch64", killCfg.Arch)
			rootLogger.Error(errorMsg.Error())
			return "", errorMsg
		}

	}

	removeJailerChrootDirectory(rootLogger, *jailingFcConfig)

	result := fmt.Sprintf("VM with id: %s has been stopped", jailingFcConfig.VMMID())
	if resultAarch64 != "" {
		result += " " + resultAarch64
	}

	return result, nil
}

func removeJailerChrootDirectory(rootLogger hclog.Logger, jailingFcConfig configs.JailingFirecrackerConfig) {
	rootLogger.Info("cleaning up jail directory")
	if err := os.RemoveAll(jailingFcConfig.JailerChrootDirectory()); err != nil {
		rootLogger.Error("jail directory removal status", "error", err)
	}
}

func (instance *FireCrackerManager) stop_x86_64(socketPath string, runningPid *pid.RunningVMMPID, rootLogger hclog.Logger) error {
	fcClient := firecracker.NewClient(socketPath, nil, false)

	ok, actionErr := fcClient.CreateSyncAction(context.Background(), &models.InstanceActionInfo{
		ActionType: firecracker.String("SendCtrlAltDel"),
	})

	if actionErr != nil {
		if !strings.Contains(actionErr.Error(), "connect: connection refused") {
			errorMsg := fmt.Errorf("failed sending CtrlAltDel to the VMM, reason: %s", actionErr)
			rootLogger.Error(errorMsg.Error())
			return errorMsg
		}
		rootLogger.Info("VMM is already stopped")
	} else {

		if runningPid != nil {
			rootLogger.Info("VMM with pid, waiting for process to exit")
			rootLogger.Info(strconv.Itoa(runningPid.Pid))
			rootLogger.Info("VMM stopped with response", "response", ok)
		}
	}
	return nil
}

func (instance *FireCrackerManager) stop_aarch64(socketPath string, runningPid *pid.RunningVMMPID, rootLogger hclog.Logger) (string, error) {
	fcClient := firecracker.NewClient(socketPath, nil, false)

	stopMetaData := struct {
		ShutDown int
	}{
		ShutDown: 1,
	}

	jsonData, err := json.Marshal(stopMetaData)
	if err != nil {
		fmt.Println("Error marshaling to JSON:", err)
		return "", err
	}

	var validMetadata interface{}

	if err := json.Unmarshal(jsonData, &validMetadata); err != nil {
		return "", fmt.Errorf("cannot parse from string to json the metadata: %v", err)
	}

	_, err = fcClient.PutMmds(context.Background(), validMetadata)

	if err != nil {
		return "", fmt.Errorf("cannot send mmds  data to vm: %v", err)
	}

	process, err := os.FindProcess(runningPid.Pid)
	if err != nil {
		return "", fmt.Errorf("cannot find process with pid: %v", runningPid.Pid)
	}

	ps, err := process.Wait()
	if err != nil {
		return "", fmt.Errorf("something went wrong listenting to pid: %v, error: %s", runningPid.Pid, err.Error())
	}

	return fmt.Sprintf("Process state: %s\n", ps.String()), nil

}
