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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type CIAOConfig struct {
	UsedFile  string
	Launcher  LauncherConfig
}

type LauncherConfig struct {
	Server      string `json:"server"`
	CACert      string `json:"cacert"`
	Cert        string `json:"cert"`
	ComputeNet  string `json:"compute-net"`
	MgmtNet     string `json:"mgmt-net"`
	HardReset   bool   `json:"hard-reset"`
	DiskLimit   bool   `json:"disk-limit"`
	MemLimit    bool   `json:"mem-limit"`
	Simulation  bool   `json:"simulation"`
}

func (launcher *LauncherConfig) initLauncher(){
	launcher.Server     = "localhost"
	launcher.CACert     = "/etc/pki/ciao/CAcert-server-localhost.pem"
	launcher.Cert       = "/etc/pki/ciao/cert-client-localhost.pem"
	launcher.ComputeNet = ""
	launcher.MgmtNet    = ""
 	launcher.HardReset  = false
	launcher.DiskLimit  = true
	launcher.MemLimit = true
	launcher.Simulation = false
}

func loadConfigFile(path, filename string, ciaoConf CIAOConfig)  CIAOConfig {
	filePath := fmt.Sprintf("%s/%s", path, filename)

	if _, err := os.Stat(filePath); err == nil {
		file,err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Error on file:  %v \n %v", filePath, err)
		} else {
			json.Unmarshal(file, &ciaoConf)
			ciaoConf.UsedFile = filePath
		}
	} 
	return ciaoConf
}

func InitConfig() CIAOConfig {
	config := CIAOConfig{}
	config.Launcher.initLauncher()
	configPaths := [...]string{
		"/home/onmunoz/dev/go/dev/viper/config1",
		"/home/onmunoz/dev/go/dev/viper/config2"}
	configFile := "ciao.json"

	for _, path := range configPaths {
		config = loadConfigFile(path, configFile, config)
	}
	return config
}
