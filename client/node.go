//
// Copyright (c) 2017 Intel Corporation
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

package client

import (
	"fmt"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
)

// ChangeNodeStatus modifies the status of a node
func (client *Client) ChangeNodeStatus(nodeID string, status types.NodeStatusType) error {
	if !client.IsPrivileged() {
		return errors.New("This command is only available to admins")
	}

	nodeStatus := types.CiaoNodeStatus{Status: status}

	url, err := client.getCiaoResource("node", api.NodeV1)
	if err != nil {
		return errors.Wrap(err, "Error getting node resource")
	}

	url = fmt.Sprintf("%s/%s", url, nodeID)

	err = client.putResource(url, api.NodeV1, &nodeStatus)

	return err
}
