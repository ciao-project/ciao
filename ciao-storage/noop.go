// add build flag

package storage

import (
	"github.com/01org/ciao/ssntp/uuid"
)

type noopDriver struct{}

func (d noopDriver) CreateBlockDevice(image *string, size int) (BlockDevice, error) {
	return BlockDevice{ID: uuid.Generate().String()}, nil
}

func (d noopDriver) DeleteBlockDevice(string) error {
	return nil
}
