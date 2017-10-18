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

package main

import (
	"github.com/golang/glog"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/uuid"
)

func validateVMWorkload(req types.Workload) error {
	// FWType must be either EFI or legacy.
	if req.FWType != string(payloads.EFI) && req.FWType != payloads.Legacy {
		return types.ErrBadRequest
	}

	// Must have storage for VMs
	if len(req.Storage) == 0 {
		return types.ErrBadRequest
	}

	return nil
}

func validateContainerWorkload(req types.Workload) error {
	// we should reject anything with ImageID set, but
	// we'll just ignore it.
	if req.ImageName == "" {
		return types.ErrBadRequest
	}

	return nil
}

func (c *controller) validateWorkloadStorageSourceID(storage *types.StorageResource, tenantID string) error {
	if storage.SourceID == "" {
		// you may only use no source id with empty type
		if storage.SourceType != types.Empty {
			return types.ErrBadRequest
		}
	}

	if storage.SourceType == types.ImageService {
		_, err := c.GetImage(tenantID, storage.SourceID)
		if err != nil {
			return types.ErrBadRequest
		}
	}

	if storage.SourceType == types.VolumeService {
		_, err := c.ShowVolumeDetails(tenantID, storage.SourceID)
		if err != nil {
			return types.ErrBadRequest
		}
	}
	return nil
}

func (c *controller) validateWorkloadStorage(req types.Workload) error {
	bootableCount := 0
	for i := range req.Storage {
		// check that a workload type is specified
		if req.Storage[i].SourceType == "" {
			return types.ErrBadRequest
		}

		// you may not request a bootable empty volume.
		if req.Storage[i].Bootable && req.Storage[i].SourceType == types.Empty {
			return types.ErrBadRequest
		}

		if req.Storage[i].ID != "" {
			// validate that the id is at least valid
			// uuid4.
			_, err := uuid.Parse(req.Storage[i].ID)
			if err != nil {
				return types.ErrBadRequest
			}

			// If we have an ID we must have a type to get it from
			if req.Storage[i].SourceType != types.Empty {
				return types.ErrBadRequest
			}
		}

		err := c.validateWorkloadStorageSourceID(&req.Storage[i], req.TenantID)
		if err != nil {
			return err
		}

		if req.Storage[i].Bootable {
			bootableCount++
		}
	}

	// must be at least one bootable volume
	if req.VMType == payloads.QEMU && bootableCount == 0 {
		return types.ErrBadRequest
	}

	return nil
}

// this is probably an insufficient amount of checking.
func (c *controller) validateWorkloadRequest(req types.Workload) error {
	// ID must be blank.
	if req.ID != "" {
		glog.V(2).Info("Invalid workload request: ID is not blank")
		return types.ErrBadRequest
	}

	// we don't validate the TenantID right now - it is passed
	// in via the ciao api, and it has passed the regex input
	// validation already. there's also a conflict with ssntp's uuid.Parse()
	// function where they assume you are using a uuid4 with '-' as
	// separator, and keystone doesn't use the '-' separator for
	// uuids.

	if req.VMType == payloads.QEMU {
		err := validateVMWorkload(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid VM workload")
			return err
		}
	} else {
		err := validateContainerWorkload(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid container workload")
			return err
		}
	}

	if req.Config == "" {
		glog.V(2).Info("Invalid workload request: config is blank")
		return types.ErrBadRequest
	}

	if len(req.Storage) > 0 {
		err := c.validateWorkloadStorage(req)
		if err != nil {
			glog.V(2).Info("Invalid workload request: invalid storage")
			return err
		}
	}

	return nil
}

func (c *controller) CreateWorkload(req types.Workload) (types.Workload, error) {
	err := c.validateWorkloadRequest(req)
	if err != nil {
		return req, err
	}

	err = c.confirmTenant(req.TenantID)
	if err != nil {
		return req, err
	}

	req.ID = uuid.Generate().String()

	err = c.ds.AddWorkload(req)
	return req, err
}

func (c *controller) DeleteWorkload(tenantID string, workloadID string) error {
	return c.ds.DeleteWorkload(tenantID, workloadID)
}

func (c *controller) ShowWorkload(tenantID string, workloadID string) (types.Workload, error) {
	return c.ds.GetWorkload(tenantID, workloadID)
}

func (c *controller) ListWorkloads(tenantID string) ([]types.Workload, error) {
	return c.ds.GetWorkloads(tenantID)
}
