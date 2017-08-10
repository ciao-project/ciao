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

package main

func (c *controller) AddUser(username, pwhash string) error {
	return c.ds.AddUser(username, pwhash)
}

func (c *controller) DelUser(username string) error {
	return c.ds.DelUser(username)
}

func (c *controller) ListUsers() ([]string, error) {
	users, err := c.ds.GetUsers()
	if err != nil {
		return []string{}, err
	}

	res := make([]string, len(users))
	for i := range users {
		res[i] = users[i].Username
	}

	return res, nil
}

func (c *controller) ListUserGrants(username string) ([]string, error) {
	ui, err := c.ds.GetUserInfo(username)
	if err != nil {
		return []string{}, err
	}

	return ui.Grants, nil
}

func (c *controller) GrantUser(username, tenantID string) error {
	return c.ds.Grant(username, tenantID)
}

func (c *controller) RevokeUser(username, tenantID string) error {
	return c.ds.Revoke(username, tenantID)
}
