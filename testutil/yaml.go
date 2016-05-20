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

package testutil

// StartYaml is a sample workload Start command payload for test usage
var StartYaml = `start:
  instance_uuid: 3390740c-dce9-48d6-b83a-a717417072ce
  image_uuid: 59460b8a-5f53-4e3e-b5ce-b71fed8c7e64
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
    - type: mem_mb
      value: 1014
      mandatory: true
    - type: disk_mb
      value: 10000
      mandatory: true
  estimated_resources:
    - type: vcpus
      value: 1
    - type: mem_mb
      value: 128
    - type: disk_mb
      value: 4096
`

// CNCIStartYaml is a sample CNCI workload Start command payload for test cases
var CNCIStartYaml = `start:
  instance_uuid: fb3e089c-62bd-476c-b22a-9d6d09599306
  image_uuid: eba04826-62a5-48bd-876f-9119667b1487,
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
    - type: vcpus
      value: 4
      mandatory: true
    - type: mem_mb
      value: 4096
      mandatory: true
    - type: network_node
      value: 1
      mandatory: true
`

// PartialStartYaml is a sample minimal workload Start command payload for test cases
var PartialStartYaml = `start:
  instance_uuid: 923d1f2b-aabe-4a9b-9982-8664b0e52f93
  image_uuid: 53cdd9ef-228f-4ce1-911d-706c2b41454a
  docker_image: ubuntu/latest
  fw_type: efi
  persistence: host
  vm_type: qemu
  requested_resources:
    - type: vcpus
      value: 2
      mandatory: true
`
