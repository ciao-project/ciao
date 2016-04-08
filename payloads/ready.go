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

package payloads

type Ready struct {
	NodeUUID        string `yaml:"node_uuid"`
	MemTotalMB      int    `yaml:"mem_total_mb"`
	MemAvailableMB  int    `yaml:"mem_available_mb"`
	DiskTotalMB     int    `yaml:"disk_total_mb"`
	DiskAvailableMB int    `yaml:"disk_available_mb"`
	Load            int    `yaml:"load"`
	CpusOnline      int    `yaml:"cpus_online"`
}

func (s *Ready) Init() {
	s.NodeUUID = ""
	s.MemTotalMB = -1
	s.MemAvailableMB = -1
	s.DiskTotalMB = -1
	s.DiskAvailableMB = -1
	s.Load = -1
	s.CpusOnline = -1
}
