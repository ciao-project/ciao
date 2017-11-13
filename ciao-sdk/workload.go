package sdk

import (
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/client"
	"github.com/ciao-project/ciao/payloads"

	"github.com/pkg/errors"
)

func GetWorkloadList(c *client.Client, flags CommandOpts) ([]Workload, error) {
	var workloads []Workload

	if c.TenantID == "" {
		return workloads, errors.New("Missing required TenantID parameter")
	}

	wls, err := c.ListWorkloads()
	if err != nil {
		return nil, errors.Wrap(err, "Error listing workloads")
	}

	for i, wl := range wls {
		workloads = append(workloads, Workload{
			Name: wl.Description,
			ID:   wl.ID,
		})

		for _, r := range wl.Defaults {
			if r.Type == payloads.MemMB {
				workloads[i].Mem = r.Value
			}
			if r.Type == payloads.VCPUs {
				workloads[i].CPUs = r.Value
			}
		}
	}

	return workloads, nil
}

func GetWorkload(c *client.Client, flags CommandOpts) (Workload, error) {
	var wl types.Workload

	if len(flags.Args) == 0 {
		return Workload{}, errors.New("Missing required workload UUID parameter")
	}
	workloadID := flags.Args[0]

	wl, err := c.GetWorkload(workloadID)
	if err != nil {
		return Workload{}, errors.Wrap(err, "Error getting workload")
	}

	var workload = Workload{
		Name: wl.Description,
		ID:   wl.ID,
	}
	for _, r := range wl.Defaults {
		if r.Type == payloads.MemMB {
			workload.Mem = r.Value
		}
		if r.Type == payloads.VCPUs {
			workload.CPUs = r.Value
		}
	}
	return workload, nil
}
