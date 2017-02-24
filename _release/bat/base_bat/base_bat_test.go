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

package basebat

import (
	"bytes"
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const standardTimeout = time.Second * 300
const vmCloudInit = `---
#cirros cloud-config
users:
  - name: ciaouser
    geocos: CIAO Test User
    lock-passwd: false
    passwd: abc123
    sudo: ALL=(ALL) NOPASSWD:ALL
...
`

// Get all tenants
//
// TestGetTenants calls ciao-cli tenant list -all.
//
// The test passes if the list of tenants defined for the cluster can
// be retrieved, even if the list is empty.
func TestGetTenants(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_, err := bat.GetAllTenants(ctx)
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve tenant list : %v", err)
	}
}

// Confirm that the cluster is ready
//
// Retrieve the cluster status.
//
// Cluster status is retrieved successfully, the cluster contains more than one
// node and all nodes are ready.
func TestGetClusterStatus(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	status, err := bat.GetClusterStatus(ctx)
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve cluster status : %v", err)
	}

	if status.TotalNodes == 0 {
		t.Fatalf("Cluster has no nodes")
	}

	if status.TotalNodes != status.TotalNodesReady {
		t.Fatalf("Some nodes in the cluster are not ready")
	}
}

// Get all available workloads
//
// TestGetWorkloads calls ciao-cli workload list
//
// The test passes if the list of workloads defined for the cluster can
// be retrieved, even if the list is empty.
func TestGetWorkloads(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_, err := bat.GetAllWorkloads(ctx, "")
	cancelFunc()
	if err != nil {
		t.Fatalf("Failed to retrieve workload list : %v", err)
	}
}

// Start a random workload, then make sure it's listed
//
// Retrieves the list of workloads, selects a random workload,
// creates an instance of that workload and retrieves the instance's
// status.  The instance is then deleted.
//
// The workload should be successfully launched, the status should
// be readable and the instance deleted.
func TestGetInstance(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	_, err = bat.RetrieveInstanceStatus(ctx, "", instances[0])
	if err != nil {
		t.Errorf("Failed to retrieve instance status: %v", err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Errorf("Instance %s did not launch: %v", instances[0], err)
	}

	_, err = bat.DeleteInstances(ctx, "", scheduled)
	if err != nil {
		t.Errorf("Failed to delete instance %s: %v", instances[0], err)
	}
}

// Start one instance of all workloads
//
// Retrieve list of available workloads, start one instance of each workload and
// wait for that instance to start.  Delete all started instances.
//
// The workload list should be retrieved correctly, the instances should be
// launched and achieve active status.  All instances should be deleted
// successfully.
func TestStartAllWorkloads(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	workloads, err := bat.GetAllWorkloads(ctx, "")
	if err != nil {
		t.Fatalf("Unable to retrieve workloads %v", err)
	}

	instances := make([]string, 0, len(workloads))
	for _, wkl := range workloads {
		launched, err := bat.LaunchInstances(ctx, "", wkl.ID, 1)
		if err != nil {
			t.Errorf("Unable to launch instance for workload %s : %v",
				wkl.ID, err)
			continue
		}
		scheduled, err := bat.WaitForInstancesLaunch(ctx, "", launched, true)
		if err != nil {
			t.Errorf("Instance %s did not launch correctly : %v", launched[0], err)
		}
		instances = append(instances, scheduled...)
	}

	_, err = bat.DeleteInstances(ctx, "", instances)
	if err != nil {
		t.Errorf("Failed to delete instances: %v", err)
	}
}

// Start a random workload, then get CNCI information
//
// Start a random workload and verify that there is at least one CNCI present, and that
// a CNCI exists for the tenant of the instance that has just been created.
//
// The instance should be started correctly, at least one CNCI should be returned and
// we should have a CNCI servicing the tenant to which our instance belongs.
func TestGetCNCIs(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	defer func() {
		scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
		if err != nil {
			t.Errorf("Instance %s did not launch: %v", instances[0], err)
		}

		_, err = bat.DeleteInstances(ctx, "", scheduled)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}()

	CNCIs, err := bat.GetCNCIs(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve CNCIs: %v", err)
	}

	if len(CNCIs) == 0 {
		t.Fatalf("No CNCIs found")
	}

	instanceDetails, err := bat.GetInstance(ctx, "", instances[0])
	if err != nil {
		t.Fatalf("Unable to retrieve instance[%s] details: %v",
			instances[0], err)
	}

	foundTenant := false
	for _, v := range CNCIs {
		if v.TenantID == instanceDetails.TenantID {
			foundTenant = true
			break
		}
	}

	if !foundTenant {
		t.Fatalf("Unable to locate a CNCI for instance[%s]", instances[0])
	}
}

// Start 3 random instances and make sure they're all listed
//
// Start 3 random instances, wait for them to be scheduled and retrieve
// their details.  Check some of the instance fields to ensure they're
// valid.  Finally, delete the instances.
//
// Instances should be created and scheduled.  Information about the
// instances should be successfully retrieved and this information should
// contain valid fields.
func TestGetAllInstances(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 3)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
	defer func() {
		_, err := bat.DeleteInstances(ctx, "", scheduled)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}()
	if err != nil {
		t.Fatalf("Instance %s did not launch: %v", instances[0], err)
	}

	instanceDetails, err := bat.GetAllInstances(ctx, "")
	if err != nil {
		t.Fatalf("Failed to retrieve instances: %v", err)
	}

	for _, instance := range instances {
		instanceDetail, ok := instanceDetails[instance]
		if !ok {
			t.Fatalf("Failed to retrieve instance %s", instance)
		}

		// Check some basic information

		if instanceDetail.FlavorID == "" || instanceDetail.HostID == "" ||
			instanceDetail.TenantID == "" || instanceDetail.MacAddress == "" ||
			instanceDetail.PrivateIP == "" {
			t.Fatalf("Instance missing information: %+v", instanceDetail)
		}
	}
}

// Start a random workload, then delete it
//
// Start a random instance and wait for the instance to be scheduled.  Delete all
// the instances in the current tenant and then retrieve the list of all instances.
//
// The instance should be started and scheduled, the DeleteAllInstances command should
// succeed and GetAllInstances command should return 0 instances.
func TestDeleteAllInstances(t *testing.T) {
	const retryCount = 5

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	instances, err := bat.StartRandomInstances(ctx, "", 1)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}

	_, err = bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Errorf("Instance %s did not launch: %v", instances[0], err)
	}

	err = bat.DeleteAllInstances(ctx, "")
	if err != nil {
		t.Fatalf("Failed to delete all instances: %v", err)
	}

	// TODO:  The correct thing to do here is to wait for the Delete Events
	// But these aren't correctly reported yet, see
	// https://github.com/01org/ciao/issues/792

	var i int
	var instancesFound int
	for ; i < retryCount; i++ {
		instanceDetails, err := bat.GetAllInstances(ctx, "")
		if err != nil {
			t.Fatalf("Failed to retrieve instances: %v", err)
		}

		instancesFound = len(instanceDetails)
		if instancesFound == 0 {
			break
		}

		time.Sleep(time.Second)
	}

	if instancesFound != 0 {
		t.Fatalf("0 instances expected.  Found %d", instancesFound)
	}
}

//Launch an instance created from an image with a customized cloud init
//
//Download cirros image, add it to the image service using a custom cloudinit file
//create a workload, launch an instance of this workload and connect via SSH
//using the custom user and password
//
//instance should accept connection with specified user and password and
//reply with the user name
func TestLaunchCustomInstance(t *testing.T) {
	//creating a new cirros image
	const name = "cirros-image"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	// TODO:  The only options currently supported by the image service are
	// ID and Name.  This code needs to be updated when the image service's
	// support for meta data improves.
	options := bat.ImageOptions{
		Name: name,
	}
	//downaload cirros image
	ImageFile, err := os.Create("cirros-0.3.4-x86_64-disk.img")
	if err != nil {
		t.Fatalf("Error while creating image file %v", err)
	}
	defer ImageFile.Close()

	resp, err := http.Get("http://download.cirros-cloud.net/0.3.4/cirros-0.3.4-x86_64-disk.img")
	if err != nil {
		t.Fatalf("Error while downloading image file %v", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(ImageFile, resp.Body)

	if err != nil {
		t.Fatalf("Error while saving image file %v", err)
	}

	img, err := bat.AddImage(ctx, "", "cirros-0.3.4-x86_64-disk.img", &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	defer deleteCustomData(ctx, img.ID)

	if img.ID == "" || img.Name != name || img.Status != "active" ||
		img.Visibility != "public" || img.Protected {
		t.Errorf("Meta data of added image is incorrect")
	}
	//get data to create workload yaml file
	defaults := bat.DefaultResources{
		VCPUs: 2,
		MemMB: 128,
	}

	source := bat.Source{
		Type: "image",
		ID:   img.ID,
	}
	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
	}

	WLopts := bat.WorkloadOptions{
		Description:     "BAT VM Test",
		VMType:          "qemu",
		FWType:          "legacy",
		Defaults:        defaults,
		CloudConfigFile: "CirrosInit.yaml",
		Disks:           []bat.Disk{disk},
	}
	y, err := yaml.Marshal(WLopts)
	if err != nil {
		t.Fatal("fail to read yaml file $v", err)
	}
	ioutil.WriteFile("CirrosWorkload.yaml", y, 0644)
	ioutil.WriteFile("CirrosInit.yaml", []byte(vmCloudInit), 0644)

	//create the workload
	args := []string{"workload", "create", "-yaml", "CirrosWorkload.yaml"}
	_, err = bat.RunCIAOCLIAsAdmin(ctx, "", args)
	if err != nil {
		t.Fatal("Fail to create workload $v", err)

	}
	//Get ID of workload just created
	CurWL, err := bat.GetWorkloadByName(ctx, "", WLopts.Description)
	if err != nil {
		t.Fatalf("Can't find data for workload %v", err)
	}
	//Launch instance of that workload
	instance, err := bat.LaunchInstances(ctx, "", CurWL.ID, 1)
	if err != nil {
		t.Fatal("Fail to launch instance: $v", err)
	}
	_, err = bat.WaitForInstancesLaunch(ctx, "", instance, false)
	if err != nil {
		t.Errorf("Instance %s did not launch: %v", instance[0], err)
	}
	//Access the instances with the user and pasword specified in the could init file
	config := &ssh.ClientConfig{
		User: "ciaouser",
		Auth: []ssh.AuthMethod{
			ssh.Password("abc123"),
		},
	}
	Template := `{{$x := filterContains . "Status" "active"}}{{range $x}}{{.SSHIP}}:{{.SSHPort}}{{end}}`
	argsInstance := []string{"instance", "list", "-f", Template}
	InstanceIP, err := bat.RunCIAOCLIAsAdmin(ctx, "", argsInstance)
	if err != nil {
		t.Fatal("Fail to create get Instance's data $v", err)
	}

	client, err := ssh.Dial("tcp", string(InstanceIP[:]), config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	session, err := client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("/usr/bin/whoami"); err != nil {
		panic("Failed to run: " + err.Error())
	}

	//Delete cirros-0.3.4-x86_64-disk.img and yaml files
	if _, err := os.Stat("CirrosWorkload.yaml"); err == nil {
		os.Remove("CirrosWorkload.yaml")
	}

	if _, err := os.Stat("CirrosInit.yaml"); err == nil {
		os.Remove("CirrosInit.yaml")
	}
	if _, err := os.Stat("cirros-0.3.4-x86_64-disk.img"); err == nil {
		os.Remove("cirros-0.3.4-x86_64-disk.img")
	}
	//Delete all instances
	_, err = bat.DeleteInstances(ctx, "", instance)
	if err != nil {
		t.Errorf("Failed to delete instance: %v", err)
	}

	//Delete Image
	err = bat.DeleteImage(ctx, "", img.ID)
	if err != nil {
		t.Fatalf("Unable to delete image %v", err)
	}

	//Pending delete workload until function implemented
}

func deleteCustomData(ctx context.Context, ID string) {
	_ = bat.DeleteImage(ctx, "", ID)
	//Delete cirros-0.3.4-x86_64-disk.img and yaml files
	if _, err := os.Stat("CirrosWorkload.yaml"); err == nil {
		os.Remove("CirrosWorkload.yaml")
	}

	if _, err := os.Stat("CirrosInit.yaml"); err == nil {
		os.Remove("CirrosInit.yaml")
	}
	if _, err := os.Stat("cirros-0.3.4-x86_64-disk.img"); err == nil {
		os.Remove("cirros-0.3.4-x86_64-disk.img")
	}

}

// TestMain ensures that all instances have been deleted when the tests finish.
// The individual tests do try to clean up after themselves but there's always
// the chance that a bug somewhere in ciao could lead to something not getting
// deleted.
func TestMain(m *testing.M) {
	flag.Parse()
	err := m.Run()

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	_ = bat.DeleteAllInstances(ctx, "")
	cancelFunc()

	os.Exit(err)
}
