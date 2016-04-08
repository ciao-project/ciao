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
	"encoding/gob"
	"fmt"
	"os"
	"path"

	"github.com/01org/ciao/networking/libsnnet"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"

	"github.com/golang/glog"
)

type startError struct {
	err  error
	code payloads.StartFailureReason
}

func (se *startError) send(client *ssntpConn, instance string) {
	if !client.isConnected() {
		return
	}

	payload, err := generateStartError(instance, se)
	if err != nil {
		glog.Errorf("Unable to generate payload for start_failure: %v", err)
		return
	}

	_, err = client.SendError(ssntp.StartFailure, payload)
	if err != nil {
		glog.Errorf("Unable to send start_failure: %v", err)
	}
}

func ensureBackingImage(vm virtualizer) error {

	err := vm.checkBackingImage()
	if err == errImageNotFound {
		glog.Infof("Backing image not found.  Trying to download")
		err = vm.downloadBackingImage()
		if err != nil {
			//BUG(markus): Need to change overseer state here to Downloading
			glog.Errorf("Unable to download backing image: %v", err)
			return err
		}
	} else if err != nil {
		glog.Errorf("Backing image check failed")
		return err
	}

	return nil
}

func createInstance(vm virtualizer, instanceDir string, cfg *vmConfig, bridge string, userData, metaData []byte) (err error) {
	err = os.MkdirAll(instanceDir, 0755)
	if err != nil {
		glog.Errorf("Cannot create instance directory for VM: %v", err)
		return
	}

	var cfgFile *os.File
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			_ = os.RemoveAll(instanceDir)
			if cfgFile != nil {
				_ = cfgFile.Close()
			}
		}
	}()

	err = vm.createImage(bridge, userData, metaData)
	if err != nil {
		glog.Errorf("Unable to create image %v", err)
		panic(err)
	}

	cfgFilePath := path.Join(instanceDir, instanceState)
	cfgFile, err = os.OpenFile(cfgFilePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		glog.Errorf("Unable to create state file %v", err)
		panic(err)
	}

	enc := gob.NewEncoder(cfgFile)
	err = enc.Encode(cfg)
	if err != nil {
		glog.Errorf("Failed to store state information %v", err)
		panic(err)
	}

	err = cfgFile.Close()
	cfgFile = nil
	if err != nil {
		glog.Errorf("Failed to store state information %v", err)
		panic(err)
	}

	return
}

func processStart(cmd *insStartCmd, instanceDir string, vm virtualizer, client *ssntpConn) *startError {
	var err error
	var vnicName string
	var bridge string
	var vnicCfg *libsnnet.VnicConfig

	cfg := cmd.cfg

	/*
		Need to check to see if the instance exists first.  Otherwise
		if it does exist but we fail for another reason first, the instance would be
		deleted.
	*/

	_, err = os.Stat(instanceDir)
	if err == nil {
		err = fmt.Errorf("Instance %s has already been created", cfg.Instance)
		return &startError{err, payloads.InstanceExists}
	}

	if cfg.Image == "" {
		err = fmt.Errorf("No backing image specified")
		return &startError{err, payloads.InvalidData}
	}

	if networking.Enabled() {
		vnicCfg, err = createVnicCfg(cfg)
		if err != nil {
			glog.Errorf("Could not create VnicCFG: %s", err)
			return &startError{err, payloads.InvalidData}
		}
	}

	err = ensureBackingImage(vm)
	if err != nil {
		return &startError{err, payloads.ImageFailure}
	}

	if vnicCfg != nil {
		vnicName, bridge, err = createVnic(client, vnicCfg)
		if err != nil {
			return &startError{err, payloads.NetworkFailure}
		}
	}

	err = createInstance(vm, instanceDir, cfg, bridge, cmd.userData, cmd.metaData)
	if err != nil {
		return &startError{err, payloads.ImageFailure}
	}

	err = vm.startVM(vnicName, getNodeIPAddress())
	if err != nil {
		return &startError{err, payloads.LaunchFailure}
	}

	return nil
}
