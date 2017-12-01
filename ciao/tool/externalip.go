package tool

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func ListExternalIP(c *client.Client, flags CommandOpts) ([]types.MappedIP, error) {
	IPs, err := c.ListExternalIPs()
	if err != nil {
		return nil, errors.Wrap(err, "Error listing external IPs")
	}

	return IPs, err
}
