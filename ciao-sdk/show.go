package sdk

import (
	"fmt"
	"os"

	"github.com/ciao-project/ciao/openstack/compute"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/ciao-controller/api"

	"net/http"
	"github.com/intel/tfortools"
	"github.com/spf13/cobra"
)

func listWorkloads(cmd *cobra.Command, args []string) error {
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var flavors compute.FlavorsDetails
	if *tenantID == "" {
		*tenantID = "faketenant"
	}

	url := buildComputeURL("%s/flavors/detail", *tenantID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &flavors)
	if err != nil {
		fatalf(err.Error())
	}

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "workload-list", Template,
			&flavors.Flavors, nil)
	}

	for i, flavor := range flavors.Flavors {
		fmt.Printf("Workload %d\n", i+1)
		fmt.Printf("\tName: %s\n\tUUID:%s\n\tCPUs: %d\n\tMemory: %d MB\n",
			flavor.Name, flavor.ID, flavor.Vcpus, flavor.RAM)
	}
	return nil
}

func showWorkload(cmd *cobra.Command, args[]string) error {
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

func Show(cmd *cobra.Command, args []string) {
	var ret error

	switch cmd.Use {
	case "workload":
		if len(args) == 0 {
			ret = listWorkloads(cmd, args)
		} else {
			ret = showWorkload(cmd, args)
		}
	case "event":
		if len(args) > 0 {
			fmt.Println("Event called with " + args[0])
		}
	}
	if ret != nil {
		errorf("ERROR:%s\n", ret)
	}
}