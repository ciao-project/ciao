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
	"testing"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/testutil"
)

func TestParseAttachVolumePayload(t *testing.T) {
	instance, volume, err := parseAttachVolumePayload([]byte(testutil.AttachVolumeYaml))
	if err != nil {
		t.Fatalf("parseAttachVolumePayload failed: %v", err)
	}
	if instance != testutil.InstanceUUID || volume != testutil.VolumeUUID {
		t.Fatalf("VolumeUUID or InstanceUUID is invalid")
	}

	_, _, err = parseAttachVolumePayload([]byte("  -"))
	if err == nil || err.code != payloads.AttachVolumeInvalidPayload {
		t.Fatalf("AttachVolumeInvalidPayload error expected")
	}

	_, _, err = parseAttachVolumePayload([]byte(testutil.BadAttachVolumeYaml))
	if err == nil || err.code != payloads.AttachVolumeInvalidData {
		t.Fatalf("AttachVolumeInvalidData error expected")
	}
}

func TestParseDetachVolumePayload(t *testing.T) {
	instance, volume, err := parseDetachVolumePayload([]byte(testutil.DetachVolumeYaml))
	if err != nil {
		t.Fatalf("parseDetachVolumePayload failed: %v", err)
	}
	if instance != testutil.InstanceUUID || volume != testutil.VolumeUUID {
		t.Fatalf("VolumeUUID or InstanceUUID is invalid")
	}

	_, _, err = parseDetachVolumePayload([]byte("  -"))
	if err == nil || err.code != payloads.DetachVolumeInvalidPayload {
		t.Fatalf("AttachVolumeInvalidPayload error expected")
	}

	_, _, err = parseDetachVolumePayload([]byte(testutil.BadDetachVolumeYaml))
	if err == nil || err.code != payloads.DetachVolumeInvalidData {
		t.Fatalf("DetachVolumeInvalidData error expected")
	}
}
