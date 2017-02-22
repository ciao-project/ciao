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

package service

import (
	"context"
)

type key int

// PrivKey is the index of the context map which indicates whether a
// service API has been called by a privileged user or not.
const PrivKey key = 0

// GetPrivilege returns the value of PrivKey
func GetPrivilege(ctx context.Context) bool {
	privilege, ok := ctx.Value(PrivKey).(bool)
	return privilege && ok
}

// SetPrivilege is used to set the value of PrivKey
func SetPrivilege(ctx context.Context, privileged bool) context.Context {
	return context.WithValue(ctx, PrivKey, privileged)
}
