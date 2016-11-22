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

// Package bat contains a number of helper functions that can be used to perform
// various operations on a ciao cluster such as creating an instance or retrieving
// a list of all the defined workloads, etc.  All of these helper functions are
// implemented by calling ciao-cli rather than by using ciao's REST APIs.  This
// package is mainly intended for use by BAT tests.  Manipulating the cluster
// via ciao-cli, rather than through the REST APIs, allows us to test a little
// bit more of ciao.
package bat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const instanceTemplateDesc = `{ "host_id" : "{{.HostID | js }}", 
    "tenant_id" : "{{.TenantID | js }}", "flavor_id" : "{{.Flavor.ID | js}}",
    "image_id" : "{{.Image.ID | js}}", "status" : "{{.Status | js}}",
    "ssh_ip" : "{{.SSHIP | js }}", "ssh_port" : {{.SSHPort}}
    {{ $addrLen := len .Addresses.Private }}
    {{- if gt $addrLen 0 }}
      {{- with index .Addresses.Private 0 -}}
      , "private_ip" : "{{.Addr | js }}", "mac_address" : "{{.OSEXTIPSMACMacAddr | js -}}"
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
	ID        string `json:"id"`
	Name      string `json:"name"`
	ImageUUID string `json:"disk"`
	CPUs      int    `json:"vcpus"`
	Mem       int    `json:"ram"`
}

// Instance contains detailed information about an instance
type Instance struct {
	HostID     string `json:"host_id"`
	TenantID   string `json:"tenant_id"`
	FlavorID   string `json:"flavor_id"`
	ImageID    string `json:"image_id"`
	Status     string `json:"status"`
	PrivateIP  string `json:"private_ip"`
	MacAddress string `json:"mac_address"`
	SSHIP      string `json:"ssh_ip"`
	SSHPort    int    `json:"ssh_port"`
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

// ImageOptions contains user supplied image meta data
type ImageOptions struct {
	Name             string
	ID               string
	ContainerFormat  string
	DiskFormat       string
	MinDiskGigabytes int
	MinRAMMegabytes  int
	Protected        bool
	Visibility       string
	Tags             []string
}

// Image contains all the meta data for a single image
type Image struct {
	ImageOptions
	SizeBytes   int
	Status      string
	Owner       string
	Checksum    string
	CreatedDate string
	LastUpdate  string
	File        string
	Schema      string
}

func checkEnv(vars []string) error {
	for _, k := range vars {
		if os.Getenv(k) == "" {
			return fmt.Errorf("%s is not defined", k)
		}
	}
	return nil
}

// RunCIAOCLI execs the ciao-cli command with a set of arguments.  The ciao-cli
// process will be killed if the context is Done.  An error will be returned if
// the following environment are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.  On success the data written to ciao-cli on stdout
// will be returned.
func RunCIAOCLI(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_IDENTITY", "CIAO_CONTROLLER", "CIAO_USERNAME", "CIAO_PASSWORD"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	if tenant != "" {
		args = append([]string{"-tenant", tenant}, args...)
	}

	data, err := exec.CommandContext(ctx, "ciao-cli", args...).Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao-cli %v : %v\n%s",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCLIJS is similar to RunCIAOCLI with the exception that the output
// of the ciao-cli command is expected to be in json format.  The json is
// decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCLIJS(ctx context.Context, tenant string, args []string, jsdata interface{}) error {
	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return err
	}

	return nil
}

// RunCIAOCLIAsAdmin execs the ciao-cli command as the admin user with a set of
// provided arguments.  The ciao-cli process will be killed if the context is
// Done.  An error will be returned if the following environment are not set;
// CIAO_IDENTITY,  CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
// On success the data written to ciao-cli on stdout will be returned.
func RunCIAOCLIAsAdmin(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_IDENTITY", "CIAO_CONTROLLER", "CIAO_ADMIN_USERNAME", "CIAO_ADMIN_PASSWORD"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	if tenant != "" {
		args = append([]string{"-tenant", tenant}, args...)
	}

	env := os.Environ()
	envCopy := make([]string, 0, len(env))
	for _, v := range env {
		if !strings.HasPrefix(v, "CIAO_USERNAME=") &&
			!strings.HasPrefix(v, "CIAO_PASSWORD=") {
			envCopy = append(envCopy, v)
		}
	}
	envCopy = append(envCopy, fmt.Sprintf("CIAO_USERNAME=%s",
		os.Getenv("CIAO_ADMIN_USERNAME")))
	envCopy = append(envCopy, fmt.Sprintf("CIAO_PASSWORD=%s",
		os.Getenv("CIAO_ADMIN_PASSWORD")))

	cmd := exec.CommandContext(ctx, "ciao-cli", args...)
	cmd.Env = envCopy
	data, err := cmd.Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao-cli %v : %v\n%v",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCLIAsAdminJS is similar to RunCIAOCLIAsAdmin with the exception that
// the output of the ciao-cli command is expected to be in json format.  The
// json is decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCLIAsAdminJS(ctx context.Context, tenant string, args []string,
	jsdata interface{}) error {
	data, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return err
	}

	return nil
}

// GetAllTenants retrieves a list of all tenants in the cluster by calling
// ciao-cli tenant list -all.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetAllTenants(ctx context.Context) ([]*Tenant, error) {
	var tenants []*Tenant

	args := []string{"tenant", "list", "-all", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &tenants)
	if err != nil {
		return nil, err
	}

	return tenants, nil
}

// GetAllWorkloads retrieves a list of all workloads in the cluster by calling
// ciao-cli workload list.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func GetAllWorkloads(ctx context.Context, tenant string) ([]Workload, error) {
	var workloads []Workload

	args := []string{"workload", "list", "-f", "{{tojson .}}"}
	err := RunCIAOCLIJS(ctx, tenant, args, &workloads)
	if err != nil {
		return nil, err
	}

	return workloads, nil
}

// GetInstance returns an Instance structure that contains information
// about a specific instance.  The informaion is retrieved by calling
// ciao-cli show --instance.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func GetInstance(ctx context.Context, tenant string, uuid string) (*Instance, error) {
	var instance *Instance
	args := []string{"instance", "show", "--instance", uuid, "-f", instanceTemplateDesc}
	err := RunCIAOCLIJS(ctx, tenant, args, &instance)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// GetAllInstances returns information about all instances in the specified
// tenant in a map.  The key of the map is the instance uuid.  The information
// is retrieved by calling ciao-cli instance list.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
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
	args := []string{"instance", "list", "-f", template}
	err := RunCIAOCLIJS(ctx, tenant, args, &instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// RetrieveInstanceStatus retrieve the status of a specific instance.  This
// information is retrieved using ciao-cli instance show.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func RetrieveInstanceStatus(ctx context.Context, tenant string, instance string) (string, error) {
	args := []string{"instance", "show", "-instance", instance, "-f", "{{.Status}}"}
	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RetrieveInstancesStatuses retrieves the statuses of a slice of specific instances.
// This information is retrieved using ciao-cli instance list.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
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
	args := []string{"instance", "list", "-f", template}
	err := RunCIAOCLIJS(ctx, tenant, args, &statuses)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

// DeleteInstance deletes a specific instance from the cluster.  It deletes
// the instance using ciao-cli instance delete.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
func DeleteInstance(ctx context.Context, tenant string, instance string) error {
	args := []string{"instance", "delete", "-instance", instance}
	_, err := RunCIAOCLI(ctx, tenant, args)
	return err
}

// DeleteInstances deletes a set of instances provided by the instances slice.
// If the function encounters an error deleting an instance it records the error
// and proceeds to the delete the next instance. The function returns two values,
// an error and a slice of errors.  A single error value is set if any of the
// instance deletion attempts failed. A slice of errors is also returned so that
// the caller can determine which of the deletion attempts failed. The indices
// in the error slice match the indicies in the instances slice, i.e., a non nil
// value in the first element of the error slice indicates that there was an
// error deleting the first instance in the instances slice.  An error will be
// returned if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_USERNAME, CIAO_PASSWORD.
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

// DeleteAllInstances deletes all the instances created for the specified
// tenant by calling ciao-cli instance delete -all.  It returns an error
// if the ciao-cli command fails.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func DeleteAllInstances(ctx context.Context, tenant string) error {
	args := []string{"instance", "delete", "-all"}
	_, err := RunCIAOCLI(ctx, tenant, args)
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
// scheduled.  An instance is scheduled when its status changes from pending
// to exited or active.  If mustBeActive is set to true, the function will
// fail if it sees an instance that has been scheduled but whose status is
// exited.  The function returns a slice of instance UUIDs and an error.
// In the case of success, the returned slice of UUIDs will equal the instances
// array.  In the case of error, these two slices may be different.  This
// can happen if one or more of the instances has failed to launch.  If errors
// are detected with multiple instances, e.g., mustBeActive is true and two
// instances have a status of 'exited' the error returned will refers to the
// first instance only.    An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
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

func parseInstances(data []byte, num int) ([]string, error) {
	instances := make([]string, num)
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for i := 0; i < num; i++ {
		if !scanner.Scan() {
			return nil, fmt.Errorf(
				"Missing instance UUID.  Found %d, expected %d", i, num)
		}

		line := scanner.Bytes()
		colonIndex := bytes.LastIndexByte(line, ':')
		if colonIndex == -1 || colonIndex+2 >= len(line) {
			return nil, fmt.Errorf("Unable to determine UUID of new instance")
		}
		instances[i] = string(bytes.TrimSpace(line[colonIndex+2:]))
	}

	return instances, nil
}

// LaunchInstances launches num instances of the specified workload.  On success
// the function returns a slice of UUIDs of the new instances.  The instances
// are launched using ciao-cli instance add.  An error will be returned if the
// following environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
func LaunchInstances(ctx context.Context, tenant string, workload string, num int) ([]string, error) {
	args := []string{"instance", "add", "--workload", workload,
		"--instances", fmt.Sprintf("%d", num)}
	data, err := RunCIAOCLI(ctx, tenant, args)
	if err != nil {
		return nil, err
	}

	return parseInstances(data, num)
}

// StartRandomInstances starts a specified number of instances using
// a random workload.  The UUIDs of the started instances are returned
// to the user.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_USERNAME, CIAO_PASSWORD.
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

// GetCNCIs returns a map of the CNCIs present in the cluster.  The key
// of the map is the CNCI ID.  The CNCI information is retrieved using
// ciao-cli list -cnci command.  An error will be returned if the
// following environment are not set;  CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
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
	args := []string{"node", "list", "-cnci", "-f", template}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &CNCIs)
	if err != nil {
		return nil, err
	}

	return CNCIs, nil
}

// GetClusterStatus returns the status of the ciao cluster.  The information
// is retrieved by calling ciao-cli node status.  An error will be returned
// if the following environment are not set; CIAO_IDENTITY,  CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	var cs *ClusterStatus
	args := []string{"node", "status", "-f", "{{tojson .}}"}
	err := RunCIAOCLIAsAdminJS(ctx, "", args, &cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func computeImageAddArgs(options *ImageOptions) []string {
	args := make([]string, 0, 8)

	if options.ContainerFormat != "" {
		args = append(args, "-container-format", options.ContainerFormat)
	}
	if options.DiskFormat != "" {
		args = append(args, "-disk-format", options.DiskFormat)
	}
	if options.ID != "" {
		args = append(args, "-id", options.ID)
	}
	if options.MinDiskGigabytes != 0 {
		args = append(args, "-min-disk-size",
			fmt.Sprintf("%d", options.MinDiskGigabytes))
	}
	if options.MinRAMMegabytes != 0 {
		args = append(args, "-min-ram-size",
			fmt.Sprintf("%d", options.MinRAMMegabytes))
	}
	if options.Name != "" {
		args = append(args, "-name", options.Name)
	}
	if options.Protected {
		args = append(args, "-protected")
	}
	if len(options.Tags) > 0 {
		args = append(args, "-tags", strings.Join(options.Tags, ","))
	}
	if options.Visibility != "" {
		args = append(args, "-visibility", options.Visibility)
	}

	return args
}

// AddImage uploads a new image to the ciao-image service.  The caller can
// supply a number of pieces of meta data about the image via the options
// parameter.  It is implemented by calling ciao-cli image add.
// On success the function returns the entire meta data of the
// newly updated image that includes the caller supplied meta data and the
// meta data added by the image service.  An error will be returned
// if the following environment variables are not set; CIAO_IDENTITY,
// CIAO_CONTROLLER, CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func AddImage(ctx context.Context, tenant, path string, options *ImageOptions) (*Image, error) {
	var img *Image
	args := []string{"image", "add", "-f", "{{tojson .}}", "-file", path}
	args = append(args, computeImageAddArgs(options)...)
	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &img)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// DeleteImage deletes an image from the image service.  It is implemented
// by calling ciao-cli image delete.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func DeleteImage(ctx context.Context, tenant, ID string) error {
	args := []string{"image", "delete", "-image", ID}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// GetImage retrieves the meta data for a given image.  It is implemented by
// calling ciao-cli image show.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetImage(ctx context.Context, tenant, ID string) (*Image, error) {
	var img *Image
	args := []string{"image", "show", "-image", ID, "-f", "{{tojson .}}"}

	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &img)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// GetImages retrieves the meta data for all images.  It is implemented by
// calling ciao-cli image list.  An error will be returned if the following
// environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func GetImages(ctx context.Context, tenant string) (map[string]*Image, error) {
	var images map[string]*Image
	template := `
{
{{- range $i, $val := .}}
  {{- if $i }},{{end}}
  "{{$val.ID | js }}" : {{tojson $val}}
{{- end }}
}
`
	args := []string{"image", "list", "-f", template}
	err := RunCIAOCLIAsAdminJS(ctx, tenant, args, &images)
	if err != nil {
		return nil, err
	}

	return images, nil
}

// GetImageCount returns the number of images currently stored in the
// image service.  It is implemented by calling ciao-cli image list.
// An error will be returned if the following environment variables are
// not set; CIAO_IDENTITY, CIAO_CONTROLLER, CIAO_ADMIN_USERNAME,
// CIAO_ADMIN_PASSWORD.
func GetImageCount(ctx context.Context, tenant string) (int, error) {
	args := []string{"image", "list", "-f", "{{len .}}"}

	data, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(data))
}

// UploadImage overrides the contents of an existing image with a new file.  It is
// implemented by calling ciao-cli image upload.  An error will be returned if the
// following environment variables are not set; CIAO_IDENTITY, CIAO_CONTROLLER,
// CIAO_ADMIN_USERNAME, CIAO_ADMIN_PASSWORD.
func UploadImage(ctx context.Context, tenant, ID, path string) error {
	args := []string{"image", "upload", "-image", ID, "-file", path}
	_, err := RunCIAOCLIAsAdmin(ctx, tenant, args)
	return err
}

// CreateRandomFile creates a file of the desired size with random data
// returning the path.
func CreateRandomFile(sizeMB int) (path string, err error) {
	var f *os.File
	f, err = ioutil.TempFile("/tmp", "ciao-random-")
	if err != nil {
		return
	}
	defer func() {
		err1 := f.Close()
		if err1 != nil && err == nil {
			err = err1
		}
	}()

	b := make([]byte, sizeMB*1000000)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	_, err = f.Write(b)
	if err == nil {
		path = f.Name()
	}

	return
}
