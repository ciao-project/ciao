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

package bat

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

const instanceTemplateDesc = `{ "node_id" : "{{.NodeID | js }}",
    "tenant_id" : "{{.TenantID | js }}", "workload_id" : "{{.WorkloadID | js}}",
    "status" : "{{.Status | js}}",
    "ssh_ip" : "{{.SSHIP | js }}", "ssh_port" : {{.SSHPort}},
    "volumes" : {{tojson .Volumes}}
    {{ $addrLen := len .PrivateAddresses }}
    {{- if gt $addrLen 0 }}
      {{- with index .PrivateAddresses 0 -}}
      , "private_ip" : "{{.Addr | js }}", "mac_address" : "{{.MacAddr | js -}}"
      {{end -}}
    {{- end }}
  }
`

// Tenant contains basic information about a tenant
type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Workload contains detailed information about a workload
type Workload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CPUs int    `json:"vcpus"`
	Mem  int    `json:"ram"`
}

// Instance contains detailed information about an instance
type Instance struct {
	NodeID     string   `json:"node_id"`
	TenantID   string   `json:"tenant_id"`
	WorkloadID string   `json:"workload_id"`
	Status     string   `json:"status"`
	PrivateIP  string   `json:"private_ip"`
	MacAddress string   `json:"mac_address"`
	SSHIP      string   `json:"ssh_ip"`
	SSHPort    int      `json:"ssh_port"`
	Volumes    []string `json:"volumes"`
}

// CNCI contains information about a CNCI
type CNCI struct {
	TenantID  string   `json:"tenant_id"`
	IPv4      string   `json:"ip"`
	Geography string   `json:"geo"`
	Subnets   []string `json:"subnets"`
}

// ClusterStatus contains information about the status of a ciao cluster
type ClusterStatus struct {
	TotalNodes            int `json:"total_nodes"`
	TotalNodesReady       int `json:"total_nodes_ready"`
	TotalNodesFull        int `json:"total_nodes_full"`
	TotalNodesOffline     int `json:"total_nodes_offline"`
	TotalNodesMaintenance int `json:"total_nodes_maintenance"`
}

// NodeStatus contains information about the status of a node
type NodeStatus struct {
	ID                    string    `json:"id"`
	Timestamp             time.Time `json:"updated"`
	Status                string    `json:"status"`
	MemTotal              int       `json:"ram_total"`
	MemAvailable          int       `json:"ram_available"`
	DiskTotal             int       `json:"disk_total"`
	DiskAvailable         int       `json:"disk_available"`
	Load                  int       `json:"load"`
	OnlineCPUs            int       `json:"online_cpus"`
	TotalInstances        int       `json:"total_instances"`
	TotalRunningInstances int       `json:"total_running_instances"`
	TotalPendingInstances int       `json:"total_pending_instances"`
	TotalPausedInstances  int       `json:"total_paused_instances"`
	Hostname              string    `json:"hostname"`
}

// GetUserTenants retrieves a list of all the tenants the current user has
// access to. An error will be returned if the following environment variables
// are not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetUserTenants(ctx context.Context) ([]*Tenant, error) {
	var tenants []*Tenant

	args := []string{"list", "tenants", "-f", "{{tojson .}}"}
	err := RunCIAOCmdJS(ctx, "", args, &tenants)
	if err != nil {
		return nil, err
	}

	return tenants, nil
}

// GetInstance returns an Instance structure that contains information about a
// specific instance. The information is retrieved by calling ciao show
// instance. An error will be returned if the following environment variables
// are not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetInstance(ctx context.Context, tenant string, uuid string) (*Instance, error) {
	var instance *Instance
	template := instanceTemplateDesc
	args := []string{"show", "instance", uuid, "-f", template}
	err := RunCIAOCmdJS(ctx, tenant, args, &instance)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetAllInstances returns information about all instances in the specified
// tenant in a map. The key of the map is the instance uuid. The information is
// retrieved by calling ciao list instances. An error will be returned if the
// following environment variables are not set; CIAO_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func GetAllInstances(ctx context.Context, tenant string) (map[string]*Instance, error) {
	var instances map[string]*Instance
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}}
  "{{$val.ID | js }}" : {{with $val}}` + instanceTemplateDesc + `{{end}}
{{- end }}
}
`
	args := []string{"list", "instances", "-f", template}
	err := RunCIAOCmdJS(ctx, tenant, args, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// RetrieveInstanceStatus retrieve the status of a specific instance. This
// information is retrieved using ciao show instances . An error will be
// returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func RetrieveInstanceStatus(ctx context.Context, tenant string, instance string) (string, error) {
	args := []string{"show", "instance", instance, "-f", "{{.Status}}"}
	data, err := RunCIAOCmd(ctx, tenant, args)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RetrieveInstancesStatuses retrieves the statuses of a slice of specific
// instances. This information is retrieved using ciao list instances. An
// error will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func RetrieveInstancesStatuses(ctx context.Context, tenant string) (map[string]string, error) {
	var statuses map[string]string
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}} 
   "{{$val.ID | js }}" : "{{$val.Status | js }}"
{{- end }}
}
`
	args := []string{"list", "instances", "-f", template}
	err := RunCIAOCmdJS(ctx, tenant, args, &statuses)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

// StopInstance stops a ciao instance by invoking the ciao stop instance
// command. An error will be returned if the following environment variables are
// not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func StopInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"stop", "instance", instance}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

// WaitForInstanceExit blocks until the specified instance has exited or the
// context is cancelled.  An error will be returned if the following environment
// variables are not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func WaitForInstanceExit(ctx context.Context, tenant string, instance string) error {
	for {
		status, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err != nil {
			return err
		}

		if status == "exited" {
			return nil
		}

		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// StopInstanceAndWait stops a ciao instance by invoking the ciao instance stop
// command. It then waits until the instance's status changes to exited. An
// error will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func StopInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := StopInstance(ctx, tenant, instance); err != nil {
		return err
	}
	return WaitForInstanceExit(ctx, tenant, instance)
}

// RestartInstance restarts a ciao instance by invoking the ciao restart
// instance command. An error will be returned if the following environment
// variables are not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func RestartInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"restart", "instance", instance}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

// RestartInstanceAndWait restarts a ciao instance by invoking the ciao instance
// restart command. It then waits until the instance's status changes to active.
// An error will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func RestartInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := RestartInstance(ctx, tenant, instance); err != nil {
		return err
	}
	for {
		status, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err != nil {
			return err
		}

		if status == "active" {
			return nil
		}

		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// DeleteInstance deletes a specific instance from the cluster. It deletes the
// instance using ciao delete instance. An error will be returned if the
// following environment variables are not set; CIAO_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func DeleteInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"delete", "instance", instance}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

// DeleteInstanceAndWait deletes a specific instance from the cluster. It
// deletes the instance using ciao delete instance and then blocks until ciao
// reports that the instance is truly deleted. An error will be returned if the
// following environment variables are not set; CIAO_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func DeleteInstanceAndWait(ctx context.Context, tenant string, instance string) error {
	if err := DeleteInstance(ctx, tenant, instance); err != nil {
		return err
	}

	// TODO:  The correct thing to do here is to wait for the Delete Events
	// But these do not yet contain enough information to easily identify
	// the event we're interested in.

	for {
		_, err := RetrieveInstanceStatus(ctx, tenant, instance)
		if err == nil {
			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err == context.Canceled {
			return err
		}

		return nil
	}
}

// DeleteInstances deletes a set of instances provided by the instances slice.
// If the function encounters an error deleting an instance it records the error
// and proceeds to the delete the next instance. The function returns two
// values, an error and a slice of errors. A single error value is set if any of
// the instance deletion attempts failed. A slice of errors is also returned so
// that the caller can determine which of the deletion attempts failed. The
// indices in the error slice match the indicies in the instances slice, i.e., a
// non nil value in the first element of the error slice indicates that there
// was an error deleting the first instance in the instances slice. An error
// will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func DeleteInstances(ctx context.Context, tenant string, instances []string) ([]error, error) {
	var err error
	errs := make([]error, len(instances))

	for i, instance := range instances {
		errs[i] = DeleteInstance(ctx, tenant, instance)
		if err == nil && errs[i] != nil {
			err = fmt.Errorf("At least one instance deletion attempt failed")
		}
	}

	return errs, err
}

// DeleteAllInstances deletes all the instances created for the specified tenant
// by calling ciao delete instance --all. It returns an error if the ciao
// command fails. An error will be returned if the following environment
// variables are not set; CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func DeleteAllInstances(ctx context.Context, tenant string) error {
	args := []string{"delete", "instance", "--all"}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

func checkStatuses(instances []string, statuses map[string]string,
	mustBeActive bool) ([]string, bool, error) {

	var err error
	scheduled := make([]string, 0, len(instances))
	finished := true
	for _, instance := range instances {
		status, ok := statuses[instance]
		if !ok {
			if err == nil {
				err = fmt.Errorf("Instance %s does not exist", instance)
			}
			continue
		}

		scheduled = append(scheduled, instance)

		if status == "pending" {
			finished = false
		} else if err == nil && mustBeActive && status == "exited" {
			err = fmt.Errorf("Instance %s has exited", instance)
		}
	}

	return scheduled, finished, err
}

// WaitForInstancesLaunch waits for a slice of newly created instances to be
// scheduled. An instance is scheduled when its status changes from pending to
// exited or active. If mustBeActive is set to true, the function will fail if
// it sees an instance that has been scheduled but whose status is exited. The
// function returns a slice of instance UUIDs and an error. In the case of
// success, the returned slice of UUIDs will equal the instances array. In the
// case of error, these two slices may be different. This can happen if one or
// more of the instances has failed to launch. If errors are detected with
// multiple instances, e.g., mustBeActive is true and two instances have a
// status of 'exited' the error returned will refers to the first instance only.
// An error will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func WaitForInstancesLaunch(ctx context.Context, tenant string, instances []string,
	mustBeActive bool) ([]string, error) {

	scheduled := make([]string, 0, len(instances))
	for {
		statuses, err := RetrieveInstancesStatuses(ctx, tenant)
		if err != nil {
			return scheduled, err
		}

		var finished bool
		scheduled, finished, err = checkStatuses(instances, statuses, mustBeActive)
		if finished || err != nil {
			return scheduled, err
		}

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return scheduled, ctx.Err()
		}
	}
}

// LaunchInstances launches num instances of the specified workload. On success
// the function returns a slice of UUIDs of the successfully launched instances.
// If some instances failed to start then the error can be found in the event
// log. The instances are launched using ciao create instance. If no instances
// successfully launch then an error will be returned. An error will be returned
// if the following environment variables are not set; CIAO_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func LaunchInstances(ctx context.Context, tenant string, workload string, num int) ([]string, error) {
	template := `
[
{{- range $i, $val := .}}
  {{- if $i }},{{end}}"{{$val.ID | js }}"
{{- end }}
]
`
	args := []string{"create", "instance", workload,
		"--instances", fmt.Sprintf("%d", num), "-f", template}
	var instances []string
	err := RunCIAOCmdJS(ctx, tenant, args, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// StartRandomInstances starts a specified number of instances using a random
// workload. The UUIDs of the started instances are returned to the user. An
// error will be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func StartRandomInstances(ctx context.Context, tenant string, num int) ([]string, error) {
	wklds, err := GetAllWorkloads(ctx, tenant)
	if err != nil {
		return nil, err
	}

	if len(wklds) == 0 {
		return nil, fmt.Errorf("No workloads defined")
	}

	wkldUUID := wklds[rand.Intn(len(wklds))].ID
	return LaunchInstances(ctx, tenant, wkldUUID, num)
}

// GetCNCIs returns a map of the CNCIs present in the cluster. The key of the
// map is the CNCI ID. The CNCI information is retrieved using ciao list cncis
// command. An error will be returned if the following environment variables are
// not set; CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetCNCIs(ctx context.Context) (map[string]*CNCI, error) {
	var CNCIs map[string]*CNCI
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}} 
   "{{$val.ID | js }}" : {
    "tenant_id" : "{{$val.TenantID | js }}", "ip" : "{{$val.IPv4 | js}}",
    "geo": "{{$val.Geography | js }}", "subnets": [
        {{- range $j, $net := $val.Subnets -}}
              {{- if $j }},{{end -}}
              "{{- $net.Subnet -}}"
        {{- end -}}
    ]}
  {{- end }}
}
`
	args := []string{"list", "cncis", "-f", template}
	err := RunCIAOCmdAsAdminJS(ctx, "", args, &CNCIs)
	if err != nil {
		return nil, err
	}

	return CNCIs, nil
}

func getNodes(ctx context.Context, args []string) (map[string]*NodeStatus, error) {
	var nodeList []*NodeStatus
	err := RunCIAOCmdAsAdminJS(ctx, "", args, &nodeList)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[string]*NodeStatus)
	for _, n := range nodeList {
		nodeMap[n.ID] = n
	}

	return nodeMap, nil
}

// GetComputeNode returns status information for a given node in the cluster.
// The information is retrieved using caio show node command. An error will
// be returned if the following environment variables are not set;
// CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetComputeNode(ctx context.Context, nodeID string) (*NodeStatus, error) {
	var node NodeStatus
	args := []string{"show", "node", nodeID, "-f", "{{tojson .}}"}
	err := RunCIAOCmdAsAdminJS(ctx, "", args, &node)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// WaitForComputeNodeStatus waits for the status of a node to transition to a
// given state.  An error will be returned if the following environment variables
// are not set; CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func WaitForComputeNodeStatus(ctx context.Context, nodeID, status string) error {
	for {
		node, err := GetComputeNode(ctx, nodeID)
		if err != nil {
			return err
		}

		if node.Status == status {
			return nil
		}

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetComputeNodes returns a map containing status information about each
// compute node in the cluster. The key of the map is the Node ID. The
// information is retrieved using ciao list nodes command. An error will
// be returned if the following environment variables are not set;
// CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetComputeNodes(ctx context.Context) (map[string]*NodeStatus, error) {
	args := []string{"list", "nodes", "--compute-nodes", "-f", "{{tojson .}}"}
	return getNodes(ctx, args)
}

// GetNetworkNodes returns a map containing status information about each
// network node in the cluster. The key of the map is the Node ID. The
// information is retrieved using ciao list nodes command. An error will
// be returned if the following environment variables are not set;
// CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func GetNetworkNodes(ctx context.Context) (map[string]*NodeStatus, error) {
	args := []string{"list", "nodes", "--network-nodes", "-f", "{{tojson .}}"}
	return getNodes(ctx, args)
}

// Evacuate evacuates a given node of a ciao cluster.  An error will be returned
// if the following environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func Evacuate(ctx context.Context, nodeid string) error {
	args := []string{"evacuate", nodeid}
	_, err := RunCIAOCmdAsAdmin(ctx, "", args)
	return err
}

// Restore restores a given node of a ciao cluster.  An error will be returned
// if the following environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func Restore(ctx context.Context, nodeid string) error {
	args := []string{"restore", nodeid}
	_, err := RunCIAOCmdAsAdmin(ctx, "", args)
	return err
}
