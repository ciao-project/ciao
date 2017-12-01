package sdk

import (
	"fmt"
	"os"

	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"

	"gopkg.in/yaml.v2"
	"github.com/intel/tfortools"
	"net/http"
)

func listWorkloads() error {
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var wls []types.Workload

	var url string
	if checkPrivilege() {
		url = buildCiaoURL("workloads")
	} else {
		url = buildCiaoURL("%s/workloads", *tenantID)
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.WorkloadsV1)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &wls)
	if err != nil {
		fatalf(err.Error())
	}

	var workloads []Workload
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

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "workload-list", Template,
			workloads, nil)
	}

	for i, wl := range workloads {
		fmt.Printf("Workload %d\n", i+1)
		fmt.Printf("\tName: %s\n\tUUID:%s\n\tCPUs: %d\n\tMemory: %d MB\n",
			wl.Name, wl.ID, wl.CPUs, wl.Mem)
	}

	return nil
}

func outputWorkload(w types.Workload) {
	var opt workloadOptions

	opt.Description = w.Description
	opt.VMType = string(w.VMType)
	opt.FWType = w.FWType
	opt.ImageName = w.ImageName
	for _, d := range w.Defaults {
		if d.Type == payloads.VCPUs {
			opt.Defaults.VCPUs = d.Value
		} else if d.Type == payloads.MemMB {
			opt.Defaults.MemMB = d.Value
		}
	}

	for _, s := range w.Storage {
		d := disk{
			Size:      s.Size,
			Bootable:  s.Bootable,
			Ephemeral: s.Ephemeral,
		}
		if s.ID != "" {
			d.ID = &s.ID
		}

		src := source{
			Type: s.SourceType,
			ID:   s.SourceID,
		}

		d.Source = src

		opt.Disks = append(opt.Disks, d)
	}

	b, err := yaml.Marshal(opt)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Println(string(b))
	fmt.Println(w.Config)
}

func showWorkload(args []string) error {
	var wl types.Workload

	if len(args) < 1 {
		fatalf("Error: need workload UUID")
	}
	workloadUUID := args[0]

	url, err := getCiaoWorkloadsResource()
	if err != nil {
		fatalf(err.Error())
	}

	ver := api.WorkloadsV1

	// you should do a get first and search for the workload,
	// then use the href - but not with the currently used
	// OpenStack API. Until we support GET with a ciao API,
	// just hard code the path.
	url = fmt.Sprintf("%s/%s", url, workloadUUID)

	resp, err := sendCiaoRequest("GET", url, nil, nil, ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		fatalf("Workload show failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &wl)
	if err != nil {
		fatalf(err.Error())
	}

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "workload-show", Template, &wl, nil)
	}

	outputWorkload(wl)
	return nil
}
