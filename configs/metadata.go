package configs

import (
	"encoding/json"
	"fmt"
)

// MachineConfig provides machine configuration options.
type MetadataConfig struct {
	Data string `json:"Data" description:"Data to pass to the VM"`
}

// NewMachineConfig returns a new instance of the configuration.
func NewMetadataConfig() *MetadataConfig {
	return &MetadataConfig{
		Data:      "",
	}
}

func (r *MetadataConfig) Serialize() (interface{}, error) {

	jsonData, err := json.Marshal(r)
	if err != nil {
		fmt.Println("Error marshaling to JSON:", err)
		return nil, err
	}

	var validMetadata interface{}

	if err := json.Unmarshal(jsonData, &validMetadata); err != nil {
		return nil, fmt.Errorf("cannot parse from string to json the metadata: %v", err)
	}

	return validMetadata, nil
}
