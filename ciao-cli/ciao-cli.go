//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/identity/v3/tokens"

	"github.com/01org/ciao/payloads"
)

var scopedToken string
var tenantID string

const openstackComputePort = 8774
const openstackComputeVersion = "v2.1"

type action uint8

const (
	computeActionStart action = iota
	computeActionStop
)

type scheme string

const (
	computeHTTP  scheme = "http"
	computeHTTPS scheme = "https"
)

var computeScheme = computeHTTPS

func (s *scheme) String() string {
	switch *s {
	case computeHTTP:
		return "http"
	case computeHTTPS:
		return "https"
	}

	return ""
}

func (s *scheme) Set(value string) error {
	for _, r := range strings.Split(value, ",") {
		if r == "http" {
			*s = computeHTTP
			return nil
		} else if r == "https" {
			*s = computeHTTPS
			return nil
		} else {
			return errors.New("Unknown scheme")
		}
	}

	return nil
}

func debugf(format string, args ...interface{}) {
	glog.V(2).Infof("ciao-cli DEBUG: "+format, args...)
}

func infof(format string, args ...interface{}) {
	glog.V(1).Infof("ciao-cli INFO: "+format, args...)
}

func warningf(format string, args ...interface{}) {
	glog.Warningf("ciao-cli WARNING: "+format, args...)
}

func errorf(format string, args ...interface{}) {
	glog.Errorf("ciao-cli ERROR: "+format, args...)
}

func fatalf(format string, args ...interface{}) {
	glog.Fatalf("ciao-cli FATAL: "+format, args...)
	os.Exit(1)
}

var (
	allInstances     = flag.Bool("all-instances", false, "Select all instances")
	listInstances    = flag.Bool("list-instances", false, "List all instances for a tenant or for a compute node")
	listQuotas       = flag.Bool("list-quotas", false, "List quotas status for a tenant")
	listResources    = flag.Bool("list-resources", false, "List consumed resources for a tenant for the past 15mn")
	listWorkloads    = flag.Bool("list-workloads", false, "List all workloads")
	listTenants      = flag.Bool("list-tenants", false, "List all tenants")
	listComputeNodes = flag.Bool("list-cns", false, "List all compute nodes")
	listCNCIs        = flag.Bool("list-cncis", false, "List all CNCIs")
	listLength       = flag.Int("list-length", 0, "Maximum number of items in the reponse")
	dumpCNCI         = flag.Bool("dump-cnci", false, "Dump a CNCI details")
	dumpToken        = flag.Bool("dump-token", false, "Dump keystone tokens")
	dumpTenantID     = flag.Bool("dump-tenant-id", false, "Dump tenant UUID")
	clusterStatus    = flag.Bool("cluster-status", false, "List all compute nodes")
	launchInstances  = flag.Bool("launch-instances", false, "Launch Ciao instances")
	deleteInstance   = flag.Bool("delete-instance", false, "Delete a Ciao instance")
	stopInstance     = flag.Bool("stop-instance", false, "Stop a Ciao instance")
	restartInstance  = flag.Bool("restart-instance", false, "Restart a Ciao instance")
	workload         = flag.String("workload", "", "Workload UUID")
	instances        = flag.Int("instances", 1, "Number of instances to create")
	instance         = flag.String("instance", "", "Instance UUID")
	instanceMarker   = flag.String("instance-marker", "", "Show instance list starting from the next instance after instance-marker")
	instanceOffset   = flag.Int("instance-offset", 0, "Show instance list starting from instance #instance-offset")
	tenant           = flag.String("tenant", "", "Tenant UUID")
	scope            = flag.String("scope", "service", "Scope tenant name")
	computeNode      = flag.String("cn", "", "Compute node UUID")
	cnci             = flag.String("cnci", "", "CNCI UUID")
	controllerURL    = flag.String("controller", "localhost", "Controller URL")
	computePort      = flag.Int("computeport", openstackComputePort, "Openstack Compute API port")
	identityURL      = flag.String("identity", "", "Keystone URL")
	identityUser     = flag.String("username", "nova", "Openstack Service Username")
	identityPassword = flag.String("password", "nova", "Openstack Service Username")
)

type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

type getResult struct {
	tokens.GetResult
}

func (r getResult) ExtractProject() (string, error) {
	if r.Err != nil {
		return "", r.Err
	}

	var response struct {
		Token struct {
			ValidProject Project `mapstructure:"project"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		return "", err
	}

	return response.Token.ValidProject.ID, nil
}

func getScopedToken(username string, password string, projectScope string) (string, string, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: *identityURL + "/v3/",
		Username:         username,
		Password:         password,
		DomainID:         "default",
		AllowReauth:      true,
	}

	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		errorf("Could not get AuthenticatedClient %s\n", err)
		return "", "", nil
	}

	client := openstack.NewIdentityV3(provider)
	if client == nil {
		errorf("something went wrong")
		return "", "", nil
	}

	scope := tokens.Scope{
		ProjectName: projectScope,
		DomainName:  "default",
	}
	token, err := tokens.Create(client, opt, &scope).Extract()
	if err != nil {
		errorf("Could not extract token %s\n", err)
		return "", "", nil
	}

	r := tokens.Get(client, token.ID)
	result := getResult{r}
	tenantID, err := result.ExtractProject()
	if err != nil {
		errorf("Could not extract tenant ID %s\n", err)
		return "", "", nil
	}

	debugf("Token: %s\n", spew.Sdump(result.Body))

	if *dumpToken == true {
		spew.Dump(result.Body)
	}

	infof("Got token %s for (%s, %s, %s)\n", token.ID, username, password, projectScope)

	return token.ID, tenantID, nil
}

type queryValue struct {
	name, value string
}

func buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s://%s:%d/%s/", computeScheme, *controllerURL, *computePort, openstackComputeVersion)
	return fmt.Sprintf(prefix+format, args...)
}

func sendComputeRequest(method string, url string, values []queryValue, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, os.ExpandEnv(url), body)
	if err != nil {
		return nil, err
	}

	infof("Sending %s %s\n", method, url)

	if values != nil {
		v := req.URL.Query()

		for _, value := range values {
			infof("Adding URL query %s=%s\n", value.name, value.value)
			v.Add(value.name, value.value)
		}

		req.URL.RawQuery = v.Encode()
	}

	if scopedToken != "" {
		req.Header.Add("X-Auth-Token", scopedToken)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}

	tlsConfig := &tls.Config{}
	if computeScheme == computeHTTPS {
		warningf("Skipping TLS verification for %s scheme\n", computeScheme)
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		errorf("Could not send HTTP request %s\n", err)
		return nil, err
	}

	infof("Got HTTP response (status %s)\n", resp.Status)

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errorf("Could not read the HTTP response %s\n", err)
			spew.Dump(resp.Body)
			return resp, err
		}

		return resp, fmt.Errorf("HTTP Error [%d] for [%s %s]: %s", resp.StatusCode, method, url, respBody)
	}

	return resp, err
}

func unmarshalComputeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorf("Could not read the HTTP response %s\n", err)
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		errorf("Could not unmarshal the HTTP response %s\n", err)
		return err
	}

	if glog.V(2) {
		spew.Dump(v)
	}

	return nil
}

func listAllInstances(tenant string, workload string, marker string, offset int, limit int) {
	var servers payloads.ComputeServers
	var url string

	if tenant != "" {
		url = buildComputeURL("%s/servers/detail", tenant)
	} else if workload != "" {
		url = buildComputeURL("flavors/%s/servers/detail", workload)
	}

	var values []queryValue
	if limit > 0 {
		values = append(values, queryValue{
			name:  "limit",
			value: fmt.Sprintf("%d", limit),
		})
	}

	if offset > 0 {
		values = append(values, queryValue{
			name:  "offset",
			value: fmt.Sprintf("%d", offset),
		})
	}

	if marker != "" {
		values = append(values, queryValue{
			name:  "marker",
			value: marker,
		})
	}

	resp, err := sendComputeRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for i, server := range servers.Servers {
		fmt.Printf("Instance #%d\n", i+1)
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
	}
}

func limitToString(limit int) string {
	if limit == -1 {
		return "Unlimited"
	}

	return fmt.Sprintf("%d", limit)
}

func listTenantQuotas(tenant string) {
	var resources payloads.CiaoTenantResources
	url := buildComputeURL("%s/quotas", tenant)

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &resources)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("Quotas for tenant %s:\n", resources.ID)
	fmt.Printf("\tInstances: %d | %s\n", resources.InstanceUsage, limitToString(resources.InstanceLimit))
	fmt.Printf("\tCPUs:      %d | %s\n", resources.VCPUUsage, limitToString(resources.VCPULimit))
	fmt.Printf("\tMemory:    %d | %s\n", resources.MemUsage, limitToString(resources.MemLimit))
	fmt.Printf("\tDisk:      %d | %s\n", resources.DiskUsage, limitToString(resources.DiskLimit))
}

func listTenantResources(tenant string) {
	var usage payloads.CiaoUsageHistory
	url := buildComputeURL("%s/resources", tenant)

	now := time.Now()
	values := []queryValue{
		{
			name:  "start_date",
			value: now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			name:  "end_date",
			value: now.Format(time.RFC3339),
		},
	}

	resp, err := sendComputeRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &usage)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("Usage for tenant %s:\n", tenant)
	for _, u := range usage.Usages {
		fmt.Printf("\t%v: [%d CPUs] [%d MB memory] [%d MB disk]\n", u.Timestamp, u.VCPU, u.Memory, u.Disk)
	}
}

func workloadDetail(tenant string, workload string) string {
	var flavor payloads.ComputeFlavorDetails

	url := buildComputeURL("%s/flavors/%s", tenant, workload)

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &flavor)
	if err != nil {
		fatalf(err.Error())
	}

	return fmt.Sprintf("\tName: %s\n\tUUID:%s\n\tImage UUID: %s\n\tCPUs: %d\n\tMemory: %d MB\n",
		flavor.Flavor.Name, flavor.Flavor.ID, flavor.Flavor.Disk, flavor.Flavor.Vcpus, flavor.Flavor.RAM)
}

func listTenantWorkloads(tenant string) {
	var flavors payloads.ComputeFlavors
	if tenant == "" {
		tenant = "faketenant"
	}

	url := buildComputeURL("%s/flavors", tenant)

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &flavors)
	if err != nil {
		fatalf(err.Error())
	}

	for i, flavor := range flavors.Flavors {
		fmt.Printf("Workload %d\n", i+1)
		fmt.Printf(workloadDetail(tenant, flavor.ID))
	}
}

func listAllTenants() {
	var tenants payloads.CiaoComputeTenants

	url := buildComputeURL("tenants")

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &tenants)
	if err != nil {
		fatalf(err.Error())
	}

	for i, tenant := range tenants.Tenants {
		fmt.Printf("Tenant %d\n", i+1)
		fmt.Printf("\tUUID: %s\n", tenant.ID)
		fmt.Printf("\tName: %s\n", tenant.Name)
	}
}

func listAllComputeNodes() {
	var nodes payloads.CiaoComputeNodes

	url := buildComputeURL("nodes")

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &nodes)
	if err != nil {
		fatalf(err.Error())
	}

	for i, node := range nodes.Nodes {
		fmt.Printf("Compute Node %d\n", i+1)
		fmt.Printf("\tUUID: %s\n", node.ID)
		fmt.Printf("\tStatus: %s\n", node.Status)
		fmt.Printf("\tLoad: %d\n", node.Load)
		fmt.Printf("\tAvailable/Total memory: %d/%d MB\n", node.MemAvailable, node.MemTotal)
		fmt.Printf("\tAvailable/Total disk: %d/%d MB\n", node.DiskAvailable, node.DiskTotal)
		fmt.Printf("\tTotal Instances: %d\n", node.TotalInstances)
		fmt.Printf("\t\tRunning Instances: %d\n", node.TotalRunningInstances)
		fmt.Printf("\t\tPending Instances: %d\n", node.TotalPendingInstances)
		fmt.Printf("\t\tPaused Instances: %d\n", node.TotalPausedInstances)
	}
}

func listAllCNCIs() {
	var cncis payloads.CiaoCNCIs

	url := buildComputeURL("cncis")

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &cncis)
	if err != nil {
		fatalf(err.Error())
	}

	for i, cnci := range cncis.CNCIs {
		fmt.Printf("CNCI %d\n", i+1)
		fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
		fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
		fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
		fmt.Printf("\tSubnets:\n")
		for _, subnet := range cnci.Subnets {
			fmt.Printf("\t\t%s\n", subnet.Subnet)
		}
	}
}

func dumpCNCIDetails(cnciID string) {
	var cnci payloads.CiaoCNCI

	url := buildComputeURL("cncis/%s/detail", cnciID)

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &cnci)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
	fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
	fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
	fmt.Printf("\tSubnets:\n")
	for _, subnet := range cnci.Subnets {
		fmt.Printf("\t\t%s\n", subnet.Subnet)
	}
}

func createTenantInstance(tenant string, workload string, instances int) {
	var server payloads.ComputeCreateServer
	var servers payloads.ComputeServers

	server.Server.Workload = workload
	server.Server.MaxInstances = instances
	server.Server.MinInstances = 1

	serverBytes, err := json.Marshal(server)
	if err != nil {
		fatalf(err.Error())
	}
	body := bytes.NewReader(serverBytes)

	url := buildComputeURL("%s/servers", tenant)

	resp, err := sendComputeRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance creation failed: %s", resp.Status)
	}

	err = unmarshalComputeResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for _, server := range servers.Servers {
		fmt.Printf("Created new instance: %s\n", server.ID)
	}
}

func deleteTenantInstance(tenant string, instance string) {
	url := buildComputeURL("%s/servers/%s", tenant, instance)

	resp, err := sendComputeRequest("DELETE", url, nil, nil)
	if err != nil {
		fatalf(err.Error())

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance deletion failed: %s", resp.Status)
	}

	fmt.Printf("Deleted instance: %s\n", instance)
}

func actionAllTenantInstance(tenant string, osAction string) {
	var action payloads.CiaoServersAction

	url := buildComputeURL("%s/servers/action", tenant)

	action.Action = osAction

	actionBytes, err := json.Marshal(action)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(actionBytes)

	resp, err := sendComputeRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Action %s on all instances failed: %s", osAction, resp.Status)
	}

	fmt.Printf("%s all instances for tenant %s\n", osAction, tenant)
}

func listNodeInstances(node string) {
	var servers payloads.CiaoServersStats
	url := buildComputeURL("nodes/%s/servers/detail", node)

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &servers)
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
}

func dumpClusterStatus() {
	var status payloads.CiaoClusterStatus
	url := buildComputeURL("nodes/summary")

	resp, err := sendComputeRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalComputeResponse(resp, &status)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Printf("Total Nodes %d\n", status.Status.TotalNodes)
	fmt.Printf("\tReady %d\n", status.Status.TotalNodesReady)
	fmt.Printf("\tFull %d\n", status.Status.TotalNodesFull)
	fmt.Printf("\tOffline %d\n", status.Status.TotalNodesOffline)
	fmt.Printf("\tMaintenance %d\n", status.Status.TotalNodesMaintenance)
}

const (
	osStart  = "os-start"
	osStop   = "os-stop"
	osDelete = "os-delete"
)

func startStopInstance(tenant, instance string, action action) {
	var actionBytes []byte

	switch action {
	case computeActionStart:
		actionBytes = []byte(osStart)
	case computeActionStop:
		actionBytes = []byte(osStop)
	default:
		fatalf("Unsupported action %d\n", action)
	}

	body := bytes.NewReader(actionBytes)

	url := buildComputeURL("%s/servers/%s/action", tenant, instance)

	resp, err := sendComputeRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance action failed: %s", resp.Status)
	}

	switch action {
	case computeActionStart:
		fmt.Printf("Instance %s restarted\n", instance)
	case computeActionStop:
		fmt.Printf("Instance %s stopped\n", instance)
	}
}

func main() {
	flag.Var(&computeScheme, "scheme", "Compute API URL scheme (http or https)")
	flag.Parse()

	if *identityURL != "" {
		if len(*identityUser) == 0 {
			fatalf("Missing required -user parameter")
		}

		if len(*identityPassword) == 0 {
			fatalf("Missing required -user parameter")
		}

		if len(*scope) == 0 {
			fatalf("Missing required -scope parameter")
		}

		t, id, err := getScopedToken(*identityUser, *identityPassword, *scope)
		if err != nil {
			fatalf(err.Error())
		}

		scopedToken = t
		tenantID = id
	}

	if *dumpTenantID == true {
		fmt.Printf("Tenant UUID: %s\n", tenantID)
	}

	if *listInstances == true {
		if len(*tenant) != 0 {
			listAllInstances(*tenant, "", *instanceMarker, *instanceOffset, *listLength)
		} else if len(*computeNode) != 0 {
			listNodeInstances(*computeNode)
		} else if len(*workload) != 0 {
			listAllInstances("", *workload, *instanceMarker, *instanceOffset, *listLength)
		} else {
			fatalf("Missing required -tenant or -cn or -workload parameters")
		}
	}

	if *listQuotas == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		listTenantQuotas(*tenant)
	}

	if *listResources == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		listTenantResources(*tenant)
	}

	if *listWorkloads == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		listTenantWorkloads(*tenant)
	}

	if *listTenants == true {
		listAllTenants()
	}

	if *listComputeNodes == true {
		listAllComputeNodes()
	}

	if *listCNCIs == true {
		listAllCNCIs()
	}

	if *clusterStatus == true {
		dumpClusterStatus()
	}

	if *launchInstances == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		if len(*workload) == 0 {
			fatalf("Missing required -workload parameter")
		}

		createTenantInstance(*tenant, *workload, *instances)
	}

	if *deleteInstance == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		if len(*instance) == 0 && *allInstances == false {
			fatalf("Missing required -instance parameter")
		}

		if *allInstances == false {
			deleteTenantInstance(*tenant, *instance)
		} else {
			actionAllTenantInstance(*tenant, osDelete)
		}
	}

	if *dumpCNCI == true {
		if len(*cnci) == 0 {
			fatalf("Missing required -cnci parameter")
		}

		dumpCNCIDetails(*cnci)
	}

	if *stopInstance == true || *restartInstance == true {
		if len(*tenant) == 0 {
			fatalf("Missing required -tenant parameter")
		}

		if len(*instance) == 0 {
			fatalf("Missing required -instance parameter")
		}

		action := computeActionStart
		if *stopInstance == true {
			action = computeActionStop
		}

		startStopInstance(*tenant, *instance, action)
	}
}
