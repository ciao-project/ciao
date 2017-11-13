package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func ListEvents(c *client.Client, flags CommandOpts) ([]types.CiaoEvent, error) {
	if flags.Tenant == "" {
		flags.Tenant = c.TenantID
	}

	if flags.All == false && flags.Tenant == "" {
		return nil, errors.New("Missing required --tenantID parameter")
	}

	tenantID := flags.Tenant
	if flags.All {
		tenantID = ""
	}

	events, err := c.ListEvents(tenantID)
	if err != nil {
		return events.Events, errors.Wrap(err, "Error listing events")
	}

	return events.Events, err
}
