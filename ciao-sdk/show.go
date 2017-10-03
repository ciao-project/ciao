package sdk

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"


	"github.com/ciao-project/ciao/openstack/compute"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/ciao-controller/api"

	"net/http"
	"github.com/intel/tfortools"
	"github.com/spf13/cobra"
)

type byCreated []compute.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

func dumpInstance(server *compute.ServerDetails) {
	fmt.Printf("\tUUID: %s\n", server.ID)
	fmt.Printf("\tStatus: %s\n", server.Status)
	fmt.Printf("\tPrivate IP: %s\n", server.Addresses.Private[0].Addr)
	fmt.Printf("\tMAC Address: %s\n", server.Addresses.Private[0].OSEXTIPSMACMacAddr)
	fmt.Printf("\tCN UUID: %s\n", server.HostID)
	fmt.Printf("\tImage UUID: %s\n", server.Image.ID)
	fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
	if server.SSHIP != "" {
		fmt.Printf("\tSSH IP: %s\n", server.SSHIP)
		fmt.Printf("\tSSH Port: %d\n", server.SSHPort)
	}

	for _, vol := range server.OsExtendedVolumesVolumesAttached {
		fmt.Printf("\tVolume: %s\n", vol)
	}
}

func listInstances(cmd *cobra.Command, args []string) error {
	if InstanceFlags.Tenant == "" {
		InstanceFlags.Tenant = *tenantID
	}

	if InstanceFlags.Computenode != "" {
		//return listNodeInstances(InstanceFlags.computenode)
	}

	var servers compute.Servers

	url := buildComputeURL("%s/servers/detail", InstanceFlags.Tenant)

	var values []queryValue
	if InstanceFlags.Limit > 0 {
		values = append(values, queryValue{
			name:  "limit",
			value: fmt.Sprintf("%d", InstanceFlags.Limit),
		})
	}

	if InstanceFlags.Offset > 0 {
		values = append(values, queryValue{
			name:  "offset",
			value: fmt.Sprintf("%d", InstanceFlags.Offset),
		})
	}

	if InstanceFlags.Marker != "" {
		values = append(values, queryValue{
			name:  "marker",
			value: InstanceFlags.Marker,
		})
	}

	if InstanceFlags.Workload != "" {
		values = append(values, queryValue{
			name:  "flavor",
			value: InstanceFlags.Workload,
		})
	}

	resp, err := sendHTTPRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	sortedServers := []compute.ServerDetails{}
	for _, v := range servers.Servers {
		sortedServers = append(sortedServers, v)
	}
	sort.Sort(byCreated(sortedServers))

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "instance-list", Template,
			&sortedServers, nil)
	}

	w := new(tabwriter.Writer)
	if !InstanceFlags.Detail {
		w.Init(os.Stdout, 0, 1, 1, ' ', 0)
		fmt.Fprintln(w, "#\tUUID\tStatus\tPrivate IP\tSSH IP\tSSH PORT")
	}

	for i, server := range sortedServers {
		if !InstanceFlags.Detail {
			fmt.Fprintf(w, "%d", i+1)
			fmt.Fprintf(w, "\t%s", server.ID)
			fmt.Fprintf(w, "\t%s", server.Status)
			fmt.Fprintf(w, "\t%s", server.Addresses.Private[0].Addr)
			if server.SSHIP != "" {
				fmt.Fprintf(w, "\t%s", server.SSHIP)
				fmt.Fprintf(w, "\t%d\n", server.SSHPort)
			} else {
				fmt.Fprintf(w, "\tN/A")
				fmt.Fprintf(w, "\tN/A\n")
			}
			w.Flush()
		} else {
			fmt.Printf("Instance #%d\n", i+1)
			dumpInstance(&server)
		}
	}
	return nil
}

func showInstance(cmd *cobra.Command, args []string) error {

	return nil
}

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
	command := strings.Fields(cmd.Use)

	switch command[0] {
	case "instance":
		ret = listInstances(cmd, args)
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
