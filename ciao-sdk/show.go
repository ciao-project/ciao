package sdk

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/openstack/compute"
	"github.com/ciao-project/ciao/payloads"

	"github.com/intel/tfortools"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"net/http"
)

type byCreated []compute.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

func listEvent(cmd *cobra.Command, args []string) error {
	var events types.CiaoEvents
	var url string

	if len(args) == 0 {
		url = buildComputeURL("events")
	} else {
		url = buildComputeURL("%s/events", args[0])
	}

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &events)
	if err != nil {
		fatalf(err.Error())
	}

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "event-list", Template,
			&events.Events, nil)
	}

	fmt.Printf("%d Ciao event(s):\n", len(events.Events))
	for i, event := range events.Events {
		fmt.Printf("\t[%d] %v: %s:%s (Tenant %s)\n", i+1, event.Timestamp, event.EventType, event.Message, event.TenantID)
	}
	return nil
}

func dumpInstance(server *compute.ServerDetails) {
	fmt.Printf("\tUUID: %s\n", server.ID)
	fmt.Printf("\tStatus: %s\n", server.Status)
	fmt.Printf("\tPrivate IP: %s\n", server.PrivateAddresses[0].Addr)
	fmt.Printf("\tMAC Address: %s\n", server.PrivateAddresses[0].MacAddr)
	fmt.Printf("\tCN UUID: %s\n", server.NodeID)
	fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
	if server.SSHIP != "" {
		fmt.Printf("\tSSH IP: %s\n", server.SSHIP)
		fmt.Printf("\tSSH Port: %d\n", server.SSHPort)
	}

	for _, vol := range server.Volumes {
		fmt.Printf("\tVolume: %s\n", vol)
	}
}

func listNodeInstances(node string) error {
	if node == "" {
		fatalf("Missing required -cn parameter")
	}

	var servers types.CiaoServersStats
	url := buildComputeURL("nodes/%s/servers/detail", node)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for i, server := range servers.Servers {
		fmt.Printf("Instance #%d\n", i+1)
		fmt.Printf("\tUUID: %s\n", server.ID)
		fmt.Printf("\tStatus: %s\n", server.Status)
		fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
		fmt.Printf("\tIPv4: %s\n", server.IPv4)
		fmt.Printf("\tCPUs used: %d\n", server.VCPUUsage)
		fmt.Printf("\tMemory used: %d MB\n", server.MemUsage)
		fmt.Printf("\tDisk used: %d MB\n", server.DiskUsage)
	}

	return nil
}

func listInstances(cmd *cobra.Command, args []string) error {
	if InstanceFlags.Tenant == "" {
		InstanceFlags.Tenant = *tenantID
	}

	if InstanceFlags.Computenode != "" {
		return listNodeInstances(InstanceFlags.Computenode)
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
			fmt.Fprintf(w, "\t%s", server.PrivateAddresses[0].Addr)
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
	instance := args[0]

	var server compute.Server
	url := buildComputeURL("%s/servers/%s", *tenantID, instance)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	err = unmarshalHTTPResponse(resp, &server)
	if err != nil {
		fatalf(err.Error())
	}

	if Template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "instance-show", Template,
			&server.Server, nil)
	}

	dumpInstance(&server.Server)
	return nil
}

func listWorkloads(cmd *cobra.Command, args []string) error {
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

func showWorkload(cmd *cobra.Command, args []string) error {
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
		if len(args) == 0 {
			ret = listInstances(cmd, args)
		} else {
			showInstance(cmd, args)
		}
	case "workload":
		if len(args) == 0 {
			ret = listWorkloads(cmd, args)
		} else {
			ret = showWorkload(cmd, args)
		}
	case "event":
		listEvent(cmd, args)
	}
	if ret != nil {
		errorf("ERROR:%s\n", ret)
	}
}
