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

package configuration

import (
	"errors"

	"github.com/01org/ciao/payloads"
)

// TODO: implement etcd struct
type etcd struct {
}

func (e *etcd) fetchConfiguration(uriStr string) (conf payloads.Configure, err error) {
	return conf, errors.New("Not implemented")
}

func (e *etcd) storeConfiguration(payloads.Configure) error {
	//empty for now
	return nil
}
