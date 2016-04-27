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
	"io/ioutil"
	"os"
	"path"
	"syscall"
)

func loadConfigFile(confPath, filename string, ciaoConf interface{}) error {
	filePath := path.Join(confPath, filename)	
	file,err := ioutil.ReadFile(filePath)
	if err != nil {
		err, ok := err.(*os.PathError)
		if ok && err.Err == syscall.ENOENT {
			return nil
		} else {
			return err
		}
	}
	err = json.Unmarshal(file, &ciaoConf)
	if err != nil {
		return err
	}
	return nil
}

func InitConfig(config interface{}) error {
	configPaths := [...]string{
		"/usr/share/defaults/ciao",
		"/etc/ciao"}
	configFile := "ciao.json"

	for _, path := range configPaths {
		err := loadConfigFile(path, configFile, &config)
		if err != nil {
			return err
		}
	}
	return nil
}
