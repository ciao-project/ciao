// Copyright (c) 2017 Intel Corporation
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

package workloadbat

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/ciao-project/ciao/bat"
)

const standardTimeout = time.Second * 300

const vmCloudInit = `---
#cloud-config
users:
  - name: demouser
    geocos: CIAO Demo User
    lock-passwd: false
    passwd: %s
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
      - %s
...
`
const dockerCloudInit = `---
#cloud-config
runcmd:
- [ /bin/bash, -c, "while true; do sleep 60; done" ]
...
`

const vmWorkloadImageName = "ubuntu-server-16.04"
const vmPublicWorkloadImageName = "clear-linux-latest"

func getWorkloadSource(ctx context.Context, t *testing.T, public bool) bat.Source {
	source := bat.Source{
		Type:   "image",
		Source: vmWorkloadImageName,
	}

	if public {
		source.Source = vmPublicWorkloadImageName
	} else {
		source.Source = vmWorkloadImageName
	}

	return source
}

func testCreateWorkload(t *testing.T, public bool) {
	// we'll use empty string for now
	tenant := ""

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	// generate ssh test keys?

	source := getWorkloadSource(ctx, t, public)

	// fill out the opt structure for this workload.
	requirements := bat.WorkloadRequirements{
		VCPUs: 2,
		MemMB: 128,
	}

	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
	}

	opt := bat.WorkloadOptions{
		Description:  "BAT VM Test",
		VMType:       "qemu",
		FWType:       "legacy",
		Requirements: requirements,
		Disks:        []bat.Disk{disk},
	}

	var ID string
	var err error
	if public {
		ID, err = bat.CreatePublicWorkload(ctx, tenant, opt, vmCloudInit)
	} else {
		ID, err = bat.CreateWorkload(ctx, tenant, opt, vmCloudInit)
	}

	if err != nil {
		t.Fatal(err)
	}

	// now retrieve the workload from controller.
	w, err := bat.GetWorkloadByID(ctx, "", ID)
	if err != nil {
		t.Fatal(err)
	}

	if w.Name != opt.Description || w.CPUs != opt.Requirements.VCPUs || w.Mem != opt.Requirements.MemMB {
		t.Fatalf("Workload not defined correctly")
	}

	// delete the workload.
	if public {
		err = bat.DeletePublicWorkload(ctx, w.ID)
	} else {
		err = bat.DeleteWorkload(ctx, tenant, w.ID)
	}

	if err != nil {
		t.Fatal(err)
	}

	// now try to retrieve the workload from controller.
	_, err = bat.GetWorkloadByID(ctx, "", ID)
	if err == nil {
		t.Fatalf("Workload not deleted correctly")
	}
}

// Check that a tenant workload can be created.
//
// Create a tenant workload and confirm that the workload exists.
//
// The new workload should be visible to the tenant and contain
// the correct resources and description.
func TestCreateTenantWorkload(t *testing.T) {
	testCreateWorkload(t, false)
}

// Check that a public workload can be created.
//
// Create a public workload and confirm that the workload exists.
//
// The new public workload should be visible to the tenant and contain
// the correct resources and description.
func TestCreatePublicWorkload(t *testing.T) {
	testCreateWorkload(t, true)
}

func findQuota(qds []bat.QuotaDetails, name string) *bat.QuotaDetails {
	for i := range qds {
		if qds[i].Name == name {
			return &qds[i]
		}
	}
	return nil
}

// Check workload creation with a sized volume.
//
// Create a workload with a storage specification that has a size, boot
// an instance from that workload and check that the storage usage goes
// up. Then delete the instance and the created workload.
//
// The new workload is created successfully and the storage used by the
// instance created from the workload matches the requested size.
func TestCreateWorkloadWithSizedVolume(t *testing.T) {
	tenant := ""

	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	source := getWorkloadSource(ctx, t, false)

	requirements := bat.WorkloadRequirements{
		VCPUs: 2,
		MemMB: 128,
	}

	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
		Size:      10,
	}

	opt := bat.WorkloadOptions{
		Description:  "BAT VM Test",
		VMType:       "qemu",
		FWType:       "legacy",
		Requirements: requirements,
		Disks:        []bat.Disk{disk},
	}

	workloadID, err := bat.CreateWorkload(ctx, tenant, opt, vmCloudInit)

	if err != nil {
		t.Fatal(err)
	}

	w, err := bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err != nil {
		t.Fatal(err)
	}

	initalQuotas, err := bat.ListQuotas(ctx, tenant, "")
	if err != nil {
		t.Error(err)
	}

	instances, err := bat.LaunchInstances(ctx, tenant, w.ID, 1)
	if err != nil {
		t.Error(err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, tenant, instances, false)
	if err != nil {
		t.Errorf("Instances failed to launch: %v", err)
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenant, "")
	if err != nil {
		t.Error(err)
	}

	storageBefore := findQuota(initalQuotas, "tenant-storage-quota")
	storageAfter := findQuota(updatedQuotas, "tenant-storage-quota")

	if storageBefore == nil || storageAfter == nil {
		t.Errorf("Quota not found for storage")
	}

	before, _ := strconv.Atoi(storageBefore.Usage)
	after, _ := strconv.Atoi(storageAfter.Usage)

	if after-before < 10 {
		t.Errorf("Storage usage not increased by expected amount")
	}

	for _, i := range scheduled {
		err = bat.DeleteInstanceAndWait(ctx, "", i)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}

	err = bat.DeleteWorkload(ctx, tenant, w.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err == nil {
		t.Fatalf("Workload not deleted correctly")
	}
}

func testSchedulableWorkloadRequirements(ctx context.Context, t *testing.T, requirements bat.WorkloadRequirements, schedulable bool) {
	tenant := ""

	source := getWorkloadSource(ctx, t, false)

	disk := bat.Disk{
		Bootable:  true,
		Source:    &source,
		Ephemeral: true,
		Size:      10,
	}

	opt := bat.WorkloadOptions{
		Description:  "BAT VM Test",
		VMType:       "qemu",
		FWType:       "legacy",
		Requirements: requirements,
		Disks:        []bat.Disk{disk},
	}

	workloadID, err := bat.CreateWorkload(ctx, tenant, opt, vmCloudInit)
	if err != nil {
		t.Error(err)
	}

	w, err := bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err != nil {
		t.Error(err)
	}

	instances, err := bat.LaunchInstances(ctx, tenant, w.ID, 1)
	if err != nil {
		t.Error(err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, tenant, instances, false)
	if schedulable {
		if err != nil {
			t.Errorf("Instances failed to launch: %v", err)
		}

		if len(scheduled) != 1 {
			t.Errorf("Unexpected number of instances: %d", len(scheduled))
		}

		instance, err := bat.GetInstance(ctx, tenant, scheduled[0])
		if err != nil {
			t.Error(err)
		}

		if requirements.NodeID != "" && instance.NodeID != requirements.NodeID {
			t.Error("Instance not scheduled to correct node")
		}

	} else {
		if err == nil {
			t.Errorf("Expected instance launch to fail")
		}
	}

	for _, i := range scheduled {
		err = bat.DeleteInstanceAndWait(ctx, "", i)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}

	err = bat.DeleteWorkload(ctx, tenant, w.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Check that scheduling by requirement works if the workload
// cannot be scheduled
//
// Create a workload with a node id requirement that cannot be met
//
// The workload should be created but an instance should not be successfully
// created for that workload.
func TestCreateUnschedulableNodeIDWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	requirements := bat.WorkloadRequirements{
		VCPUs:  2,
		MemMB:  128,
		NodeID: "made-up-node-id",
	}

	testSchedulableWorkloadRequirements(ctx, t, requirements, false)
}

// Check that scheduling by requirement works if the workload
// cannot be scheduled
//
// Create a workload with a hostname requirement that cannot be met
//
// The workload should be created but an instance should not be successfully
// created for that workload.
func TestCreateUnschedulableHostnameWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	requirements := bat.WorkloadRequirements{
		VCPUs:    2,
		MemMB:    128,
		Hostname: "made-up-hostname",
	}

	testSchedulableWorkloadRequirements(ctx, t, requirements, false)
}

func getSchedulableNodeDetails(ctx context.Context) (string, string, error) {
	nodeData := []struct {
		NodeID   string `json:"id"`
		Hostname string `json:"hostname"`
	}{}

	args := []string{"node", "list", "--compute", "-f", "{{ tojson . }}"}
	err := bat.RunCIAOCLIAsAdminJS(ctx, "", args, &nodeData)

	if err != nil {
		return "", "", err
	}

	if len(nodeData) == 0 {
		return "", "", errors.New("No nodes available")
	}

	return nodeData[0].NodeID, nodeData[0].Hostname, nil
}

// Check that scheduling by requirement works if the workload
// can be scheduled on a node
//
// Create a workload with a node id requirement that can be met
//
// The workload should be created and an instance should be successfully
// created for that workload.
func TestCreateSchedulableNodeIDWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	nodeID, _, err := getSchedulableNodeDetails(ctx)
	if err != nil {
		t.Fatal(err)
	}

	requirements := bat.WorkloadRequirements{
		VCPUs:  2,
		MemMB:  128,
		NodeID: nodeID,
	}

	testSchedulableWorkloadRequirements(ctx, t, requirements, true)
}

// Check that scheduling by requirement works if the workload
// can be scheduled on a node
//
// Create a workload with a hostname requirement that can be met
//
// The workload should be created and an instance should be successfully
// created for that workload.
func TestCreateSchedulableHostnameWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	_, hs, err := getSchedulableNodeDetails(ctx)
	if err != nil {
		t.Fatal(err)
	}

	requirements := bat.WorkloadRequirements{
		VCPUs:    2,
		MemMB:    128,
		Hostname: hs,
	}

	testSchedulableWorkloadRequirements(ctx, t, requirements, true)
}

func testSchedulableContainerWorkload(ctx context.Context, t *testing.T, imageName string, requirements bat.WorkloadRequirements, schedulable bool) {
	tenant := ""

	opt := bat.WorkloadOptions{
		Description:  "BAT Docker Test",
		ImageName:    imageName,
		VMType:       "docker",
		Requirements: requirements,
	}

	workloadID, err := bat.CreateWorkload(ctx, tenant, opt, dockerCloudInit)
	if err != nil {
		t.Error(err)
	}

	w, err := bat.GetWorkloadByID(ctx, tenant, workloadID)
	if err != nil {
		t.Error(err)
	}

	instances, err := bat.LaunchInstances(ctx, tenant, w.ID, 1)
	if schedulable {
		if err != nil {
			t.Error(err)
		}
	} else {
		if err == nil {
			t.Errorf("Expected instance launch to fail")
		}
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, tenant, instances, false)
	if err != nil {
		t.Errorf("Instances failed to launch: %v", err)
	}

	for _, i := range scheduled {
		err = bat.DeleteInstanceAndWait(ctx, "", i)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}

	err = bat.DeleteWorkload(ctx, tenant, w.ID)
	if err != nil {
		t.Fatal(err)
	}
}

// Check that creating a privileged container is limited by permissions
//
// Create a workload with a container that should be privileged. Check that
// launching fails and then change the tenant permission to enable the
// permission and check launching succeeds.
//
// The workload should be created and without permission the launching should
// fail. With permission the launching should succeed.
func TestPriviligedWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	requirements := bat.WorkloadRequirements{
		VCPUs:      2,
		MemMB:      128,
		Privileged: true,
	}

	testSchedulableContainerWorkload(ctx, t, "debian:latest", requirements, false)

	tenants, err := bat.GetUserTenants(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenants) == 0 {
		t.Fatal("Wrong number of tenants returned")
	}

	oldcfg, err := bat.GetTenantConfig(ctx, tenants[0].ID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant config: %v", err)
	}

	cfg := oldcfg
	cfg.Permissions.PrivilegedContainers = true

	err = bat.UpdateTenant(ctx, tenants[0].ID, cfg)
	if err != nil {
		t.Fatalf("Failed to update tenant: %v", err)
	}

	defer func() {
		err := bat.UpdateTenant(ctx, tenants[0].ID, oldcfg)
		if err != nil {
			t.Fatalf("Failed to update tenant: %v", err)
		}

	}()
	requirements = bat.WorkloadRequirements{
		VCPUs:      2,
		MemMB:      128,
		Privileged: true,
	}

	testSchedulableContainerWorkload(ctx, t, "debian:latest", requirements, true)
}

// Check that launching a container works
//
// Create a workload with a container image name. Check that launching succeeds.
//
// The workload should be created and launching a container should succeed.
func TestContainerWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	requirements := bat.WorkloadRequirements{
		VCPUs: 2,
		MemMB: 128,
	}

	testSchedulableContainerWorkload(ctx, t, "debian:latest", requirements, true)
}

// Check that launching a container from a specified registry works
//
// Create a workload with a container image name from a specific registry. Check
// that launching succeeds.
//
// The workload should be created and launching a container should succeed.
func TestContainerRegistryWorkload(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	requirements := bat.WorkloadRequirements{
		VCPUs: 2,
		MemMB: 128,
	}

	testSchedulableContainerWorkload(ctx, t, "docker.io/library/ubuntu:latest", requirements, true)
}
