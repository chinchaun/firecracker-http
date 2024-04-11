package strategy

import (
	"context"
	"open-fire/configs"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/hashicorp/go-hclog"
)

// Handler names
const (
	MetadataExtractorName = "fcinit.MetadataExtractor"
)

// NewMetadataExtractorHandler returns a firecracker handler which can be used to inject state into
// a virtual machine file system prior to start.
func NewMetadataExtractorHandler(logger hclog.Logger, metadata *configs.MetadataConfig) firecracker.Handler {
	return firecracker.Handler{
		Name: MetadataExtractorName,
		Fn: func(ctx context.Context, m *firecracker.Machine) error {

			serialized, err := metadata.Serialize()

			if err != nil {
				logger.Error("error while serializing metadata", "reason", err)
				return err
			}

			m.SetMetadata(ctx, serialized)

			return nil
		},
	}
}
