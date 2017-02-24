//
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
//

// Package quotasbat is a placeholder package for the basic BAT tests.
package quotasbat

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
)

const standardTimeout = time.Second * 300

func findQuota(qds []bat.QuotaDetails, name string) *bat.QuotaDetails {
	for i := range qds {
		if qds[i].Name == name {
			return &qds[i]
		}
	}
	return nil
}

func restoreQuotas(ctx context.Context, tenantID string, origQuotas []bat.QuotaDetails, currentQuotas []bat.QuotaDetails) error {
	for i := range currentQuotas {
		qd := findQuota(origQuotas, currentQuotas[i].Name)
		if qd != nil && qd.Value != currentQuotas[i].Value {
			err := bat.UpdateQuota(ctx, "", tenantID, qd.Name, qd.Value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Test getting and setting quotas
//
// Tests retrieving and setting quotas.
//
// Gets the current quotas, sets several, gets them again checks they've
// changed, restores the original and checks the restoration.
func TestQuotas(t *testing.T) {
	qn := "tenant-vcpu-quota"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	tenants, err := bat.GetUserTenants(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenants) < 1 {
		t.Fatal("Expected user to have access to at least one tenant")
	}

	tenantID := tenants[0].ID
	origQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Fatal(err)
	}

	err = bat.UpdateQuota(ctx, "", tenantID, qn, "10")
	if err != nil {
		t.Fatal(err)
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Error(err)
	}

	qd := findQuota(updatedQuotas, qn)
	if qd.Value != "10" {
		t.Error("Quota not expected value")
	}

	err = restoreQuotas(ctx, tenantID, origQuotas, updatedQuotas)
	if err != nil {
		t.Fatal(err)
	}
}

func getUsage(qds []bat.QuotaDetails, name string) int {
	qd := findQuota(qds, name)
	if qd == nil {
		return -1
	}

	value, err := strconv.Atoi(qd.Usage)
	if err != nil {
		return -1
	}

	return value
}

func checkUsage(qds []bat.QuotaDetails, wl bat.Workload, count int) bool {
	expectedMem := wl.Mem * count
	expectedCPU := wl.CPUs * count
	expectedInstances := count

	actualMem := getUsage(qds, "tenant-mem-quota")
	actualCPU := getUsage(qds, "tenant-vcpu-quota")
	actualInstances := getUsage(qds, "tenant-instances-quota")
	return actualMem == expectedMem &&
		actualCPU == expectedCPU &&
		actualInstances == expectedInstances
}

// Test reporting of instance usage
//
// Starts 3 copies of a workload and checks that the usage matches
// 3 * workload defaults
//
// Gets a workload, launches 3 instances from that workloads, gets the
// quota and usage information and then checks that the usage is as
// expected. Deletes the instances and checks the usage reflects that.
func TestInstanceUsage(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()
	tenants, err := bat.GetUserTenants(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenants) < 1 {
		t.Fatal("Expected user to have access to at least one tenant")
	}
	tenantID := tenants[0].ID

	wls, err := bat.GetAllWorkloads(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(wls) < 1 {
		t.Fatal("Expected at least one workload")
	}

	wl := wls[0]

	instances, err := bat.LaunchInstances(ctx, "", wl.ID, 3)
	if err != nil {
		t.Error(err)
	}

	scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
	if err != nil {
		t.Errorf("Instances failed to launch: %v", err)
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Error(err)
	}

	if !checkUsage(updatedQuotas, wl, 3) {
		t.Error("Usage not recorded correctly")
	}

	for _, i := range scheduled {
		err = bat.DeleteInstanceAndWait(ctx, "", i)
		if err != nil {
			t.Errorf("Failed to delete instances: %v", err)
		}
	}
	updatedQuotas, err = bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Error(err)
	}
	if !checkUsage(updatedQuotas, wl, 0) {
		t.Error("Usage not recorded correctly")
	}
}

// Workaround for https://github.com/01org/ciao/issues/1203
func launchMultipleInstances(ctx context.Context, t *testing.T, count int) error {
	wls, err := bat.GetAllWorkloads(ctx, "")
	if err != nil {
		return err
	}
	if len(wls) < 1 {
		t.Fatal("Expected at least one workload")
	}

	wl := wls[0]
	for i := 0; i < count; i++ {
		instances, err := bat.LaunchInstances(ctx, "", wl.ID, 1)
		if err != nil {
			return err
		}

		scheduled, err := bat.WaitForInstancesLaunch(ctx, "", instances, false)
		defer func() {
			_, err := bat.DeleteInstances(ctx, "", scheduled)
			if err != nil {
				t.Errorf("Failed to delete instances: %v", err)
			}
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

// Try launching instances with a quota.
//
// Sets a quota on the number of instances that can be launched and exceeds
// that.
//
// Set a quota of two instances, "tenant-instances-quota", and then try and
// start a set of three instances and check that the launch fails.
func TestInstanceLimited(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()
	tenants, err := bat.GetUserTenants(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(tenants) < 1 {
		t.Fatal("Expected user to have access to at least one tenant")
	}
	tenantID := tenants[0].ID
	origQuotas, err := bat.ListQuotas(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	err = bat.UpdateQuota(ctx, "", tenantID, "tenant-instances-quota", "2")
	if err != nil {
		t.Fatal(err)
	}

	err = launchMultipleInstances(ctx, t, 3)
	if err == nil {
		t.Errorf("Expected launch of instances to fail")
	}

	updatedQuotas, err := bat.ListQuotas(ctx, tenantID, "")
	if err != nil {
		t.Error(err)
	}

	err = restoreQuotas(ctx, tenantID, origQuotas, updatedQuotas)
	if err != nil {
		t.Error(err)
	}
}
