package tool

import (
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/pkg/errors"
)

func StartStopInstance(c *client.Client, instance string, stop bool) error {
	if c.TenantID == "" {
		return errors.New("Missing required -tenant-id parameter")
	}

	if instance == "" {
		return errors.New("Missing required -instance parameter")
	}

	if stop == true {
		err := c.StopInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error stopping instance")
		}
	} else {
		err := c.StartInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error starting instance")
		}
	}
	return nil
}

func GetInstance(c *client.Client, flags CommandOpts) (api.Server, error) {
	if len(flags.Args) == 0 {
		errors.New("Missing required -cn parameter")
	}
	instance := flags.Args[0]

	server, err := c.GetInstance(instance)
	if err != nil {
		return server, errors.Wrap(err, "Error getting instance")
	}

	return server, nil
}

func GetNodeInstances(c *client.Client, flags CommandOpts) ([]types.CiaoServerStats, error) {
	if flags.Tenant == "" {
		flags.Tenant = c.TenantID
	}

	if flags.ComputeName == "" {
		errors.New("Missing required -cn parameter")
	}

	server, err := c.ListInstancesByNode(flags.ComputeName)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting instances for node")
	}

	return server.Servers, nil
}

func GetInstances(c *client.Client, flags CommandOpts) ([]api.ServerDetails, error) {
	if flags.Tenant == "" {
		flags.Tenant = c.TenantID
	}

	servers, err := c.ListInstancesByWorkload(flags.Tenant, flags.Workload)
	if err != nil {
		return []api.ServerDetails{}, errors.Wrap(err, "Error listing instances")
	}

	Servers := []api.ServerDetails{}
	for _, v := range servers.Servers {
		Servers = append(Servers, v)
	}

	return Servers, nil
}
