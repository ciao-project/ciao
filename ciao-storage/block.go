package storage

type BlockDriver interface {
	CreateBlockDevice(image *string, sizeGB int) (BlockDevice, error)
	DeleteBlockDevice(string) error
}

type BlockDevice struct {
	ID string
}

func GetBlockDriver() BlockDriver {
	return noopDriver{}
}
