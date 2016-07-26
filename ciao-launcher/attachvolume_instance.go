/*
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
*/

package main

import (
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
)

func processAttachVolume(vm virtualizer, cfg *vmConfig, instance, instanceDir, volumeUUID string, conn serverConn) *attachVolumeError {
	if _, found := cfg.Volumes[volumeUUID]; found {
		attachErr := &attachVolumeError{nil, payloads.AttachVolumeAlreadyAttached}
		glog.Errorf("%s is already attached to attach instance %s [%s]",
			volumeUUID, instance, string(attachErr.code))
		return attachErr
	}

	cfg.Volumes[volumeUUID] = struct{}{}

	err := cfg.save(instanceDir)
	if err != nil {
		delete(cfg.Volumes, volumeUUID)
		attachErr := &attachVolumeError{err, payloads.AttachVolumeStateFailure}
		glog.Errorf("Unable to persist instance %s state [%s]: %v",
			instance, string(attachErr.code), err)
		return attachErr
	}

	return nil
}
