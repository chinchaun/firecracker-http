package response

type CreateVMResponse struct {
	IP    string `json:"ip"`
	PID   int    `json:"pid"`
	VMMiD string `json:"vmId"`
}

type ErrorResponse struct {
	ErrorMsg string `json:"error"`
}

type MountDiskResponse struct {
	MountDir string `json:"mountDir"`
}
