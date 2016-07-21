package storage

// BlockDriver is the interface that all block drivers must implement.
type BlockDriver interface {
	CreateBlockDevice(image *string, sizeGB int) (BlockDevice, error)
	DeleteBlockDevice(string) error
}

// BlockDevice contains information about a block devices.
type BlockDevice struct {
	ID string
}

// GetBlockDriver returns an implementation of the BlockDriver interface.
// For now we are just returning the noopDriver. We should probably switch
// this to take a string or just use build flags to select.
func GetBlockDriver() BlockDriver {
	return noopDriver{}
}
