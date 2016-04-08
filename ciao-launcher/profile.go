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
// +build profile

package main

import (
	"flag"
	"os"
	"runtime/pprof"

	"github.com/golang/glog"
)

var cpuProfile string

func init() {
	flag.StringVar(&cpuProfile, "cpuprofile", "", "write profile information to file")
	profileFN = func() func() {
		if cpuProfile == "" {
			return nil
		}

		f, err := os.Create(cpuProfile)
		if err != nil {
			glog.Warning("Unable to create profile file %s: %v",
				cpuProfile, err)
			return nil
		}
		pprof.StartCPUProfile(f)
		return func() {
			pprof.StopCPUProfile()
			f.Close()
		}
	}
}
