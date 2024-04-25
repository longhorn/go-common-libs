package longhorn

import (
	"strings"

	"github.com/pkg/errors"
)

// GetVolumeNameFromReplicaDataDirectoryName extracts the volume name from the replica
// data directory name.
// The replica data directory name format is expected to follow "<volume name>-<8 character hash>".
// Note: The replica data directory name is not the same as the Kubernetes Replica
// custom resource (CR) object name.
func GetVolumeNameFromReplicaDataDirectoryName(replicaName string) (string, error) {
	parts := strings.Split(replicaName, "-")
	if len(parts) > 1 && len(parts[len(parts)-1]) == 8 {
		return strings.Join(parts[:len(parts)-1], "-"), nil
	}

	return "", errors.Errorf("failed to get volume name from replica data directory name %s", replicaName)
}
