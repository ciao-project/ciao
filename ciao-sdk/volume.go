package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func GetVolumes(c *client.Client, flags CommandOpts) ([]types.Volume, error) {
	vols, err := c.ListVolumes()
	if err != nil {
		return vols, errors.Wrap(err, "Error getting volumes")
	}

	return vols, err
}

func GetVolume(c *client.Client, flags CommandOpts) (types.Volume, error) {
	if flags.Args[0] == "" {
		return types.Volume{}, errors.New("Missing required volume parameter")
	}

	vol, err := c.GetVolume(flags.Args[0])
	if err != nil {
		return vol, errors.Wrap(err, "Error getting volume")
	}

	return vol, err
}
