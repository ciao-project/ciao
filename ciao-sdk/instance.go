package sdk

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/ciao-project/ciao/openstack/compute"
	"github.com/ciao-project/ciao/ciao-controller/types"

	"github.com/intel/tfortools"
)


type byCreated []compute.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

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

func listInstances() error {
	if CommandFlags.Tenant == "" {
		CommandFlags.Tenant = *tenantID
	}

	if CommandFlags.Computenode != "" {
		return listNodeInstances(CommandFlags.Computenode)
	}

	var servers compute.Servers

	url := buildComputeURL("%s/servers/detail", CommandFlags.Tenant)

	var values []queryValue
	if CommandFlags.Limit > 0 {
		values = append(values, queryValue{
			name:  "limit",
			value: fmt.Sprintf("%d", CommandFlags.Limit),
		})
	}

	if CommandFlags.Offset > 0 {
		values = append(values, queryValue{
			name:  "offset",
			value: fmt.Sprintf("%d", CommandFlags.Offset),
		})
	}

	if CommandFlags.Marker != "" {
		values = append(values, queryValue{
			name:  "marker",
			value: CommandFlags.Marker,
		})
	}

	if CommandFlags.Workload != "" {
		values = append(values, queryValue{
			name:  "flavor",
			value: CommandFlags.Workload,
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
	if !CommandFlags.Detail {
		w.Init(os.Stdout, 0, 1, 1, ' ', 0)
		fmt.Fprintln(w, "#\tUUID\tStatus\tPrivate IP\tSSH IP\tSSH PORT")
	}

	for i, server := range sortedServers {
		if !CommandFlags.Detail {
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

func showInstance(args []string) error {
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
