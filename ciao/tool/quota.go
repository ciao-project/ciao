package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func GetQuotas(c *client.Client, flags CommandOpts) ([]types.QuotaDetails, error) {
	if flags.Tenant != "" {
		if !c.IsPrivileged() {
			return nil, errors.New("Listing quotas for other tenants is for privileged users only")
		}
	} else {
		if c.IsPrivileged() {
			return nil, errors.New("Admin user must specify the tenant with -for-tenant")
		}
	}

	quotas, err := c.ListQuotas(flags.Tenant)
	if err != nil {
		return nil, errors.Wrap(err, "Error listing quotas")
	}

	return quotas, err
}
