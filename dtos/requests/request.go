package requests

type MetadataRequest struct {
	Data string `json:"Data" description:"Data to pass to the VM"`
}

type CreateVMRequest struct {
	KernelPath       string          `json:"kernelPath"`
	RootDrivePath    string          `json:"rootDrivePath"`
	CniNetworkName   string          `json:"cniNetworkName"`
	AdditionalDrives string          `json:"additionalDrives"`
	Metadata         MetadataRequest `json:"metadata"`
	Debug            bool            `json:"debug"`
	VcpuCount        int64           `json:"vCpuCount"`
	MemSizeMib       int64           `json:"memSizeMib"`
	EnableSmt        bool            `json:"enableSmt"`
	JailerChrootBase string          `json:"jailerChrootBase"`
}

type StopVMRequest struct {
	VMMiD            string `json:"vmmId"`
	PID              int    `json:"pid"`
	Arch             string `json:"arch"`
	JailerChrootBase string `json:"jailerChrootBase"`
}

type MountDiskRequest struct {
	DiskName string `json:"diskName"`
}
