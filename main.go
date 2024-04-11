package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"open-fire/configs"
	"open-fire/dtos/requests"
	"open-fire/dtos/response"
	"open-fire/managers"
	"os"
	"strconv"
)

var (
	logConfig     = configs.NewLogginConfig()
	rootLogger    = logConfig.NewLogger(configs.LoggerOpts{Name: "http-handler"})
	machineConfig = configs.NewMachineConfig()
	killCfg       = configs.NewKillConfig()
)

func main() {

	fmt.Println("starting server")

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, "OK\n")
	})
	http.HandleFunc("/create", createRequestHandler)
	http.HandleFunc("/stop", stopRequestHandler)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	fmt.Printf("server listening on %s \n", port)

	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		errMsg := fmt.Errorf("cannot start server, reason: %s", err)
		fmt.Fprintf(os.Stdout, "%s \n", string(errMsg.Error()))
		panic(err)
	}
}

func buildCreateVMError(errorMsg string) []byte {
	resp := response.ErrorResponse{
		ErrorMsg: errorMsg,
	}
	response, err := json.Marshal(&resp)
	if err != nil {
		rootLogger.Error("failed to marshal json, %s \n", err)
		return nil
	}

	return response
}

func createRequestHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(io.Reader(r.Body))

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		response := buildCreateVMError("failed to read body " + err.Error())
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	var req requests.CreateVMRequest
	err = json.Unmarshal([]byte(body), &req)

	if err != nil {
		rootLogger.Error(err.Error())
		response := buildCreateVMError("failed to read json body")
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	err = machineConfig.WithCreateVMRequest(&req)

	if err != nil {
		response := buildCreateVMError(err.Error())
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	jailerCfg, err := configs.NewJailingFirecrackerConfigWithChrootBase(req.JailerChrootBase)

	if err != nil {
		response := buildCreateVMError(err.Error())
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	fcManager := managers.CreateFCManagerInstance()

	fcMachine, err := fcManager.StartVM(machineConfig, jailerCfg)

	if err != nil {
		response := buildCreateVMError("An error occurs when starting the vm: " + err.Error())
		w.WriteHeader(500)
		w.Write(response)
		return
	}

	pid, err := fcMachine.PID()
	if err != nil {
		rootLogger.Warn("cannot get PID: %s", err)
	}

	resp := response.CreateVMResponse{
		IP:    fcMachine.Cfg.NetworkInterfaces[0].StaticConfiguration.IPConfiguration.IPAddr.IP.String(),
		PID:   pid,
		VMMiD: fcMachine.Cfg.VMID,
	}

	response, err := json.Marshal(&resp)
	if err != nil {
		errMsg := "failed to marshal create vm response json: " + err.Error()
		response := buildCreateVMError(errMsg)
		w.WriteHeader(500)
		w.Write(response)
		return
	}

	w.WriteHeader(201)
	w.Write(response)

}

func stopRequestHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errMsg := "failed to read body: " + err.Error()
		rootLogger.Error(errMsg)
		response := buildCreateVMError(errMsg)
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	var req requests.StopVMRequest

	json.Unmarshal([]byte(body), &req)
	if err != nil {
		rootLogger.Error(err.Error())
		response := buildCreateVMError("Cannot ready body of request")
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	if req.Arch == "" || req.PID == 0 || req.VMMiD == "" {
		response := buildCreateVMError(fmt.Sprintf("missing required field, arch: %s, pid: %s, vmmid: %s", req.Arch, strconv.Itoa(req.PID), req.VMMiD))
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	fcManager := managers.CreateFCManagerInstance()

	err = killCfg.WithStopVMRequest(&req)

	if err != nil {
		response := buildCreateVMError(err.Error())
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	jailerCfg, err := configs.NewJailingFirecrackerConfigWithChrootBase(req.JailerChrootBase)

	if err != nil {
		response := buildCreateVMError(err.Error())
		w.WriteHeader(422)
		w.Write(response)
		return
	}

	result, err := fcManager.StopVM(killCfg, jailerCfg)
	if err != nil {
		errMsg := "An error occurred while stopping Firecracker VMM: " + err.Error()
		rootLogger.Error(errMsg)

		response := buildCreateVMError(errMsg)
		w.WriteHeader(500)
		w.Write(response)
		return
	}

	w.WriteHeader(201)
	io.WriteString(w, result+"\n")

}
