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

package bat

import "context"

// ExternalIP contains information about a single external ip.
type ExternalIP struct {
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`
	InstanceID string `json:"instance_id"`
	TenantID   string `json:"tenant_id"`
	PoolName   string `json:"pool_name"`
}

// CreateExternalIPPool creates a new pool for external ips. The pool is created
// using the ciao create pool command. An error will be returned if the
// following environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func CreateExternalIPPool(ctx context.Context, tenant, name string) error {
	args := []string{"create", "pool", name}
	_, err := RunCIAOCmdAsAdmin(ctx, tenant, args)
	return err
}

// AddExternalIPToPool adds an external ips to an existing pool. The address is
// added using the ciao add external-ip command. An error will be returned if the
// following environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func AddExternalIPToPool(ctx context.Context, tenant, name, ip string) error {
	args := []string{"add", "external-ip", name, ip}
	_, err := RunCIAOCmdAsAdmin(ctx, tenant, args)
	return err
}

// MapExternalIP maps an external ip from a given pool to an instance. The
// address is mapped using the ciao attach external-ip command. An error will
// be returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func MapExternalIP(ctx context.Context, tenant, pool, instance string) error {
	args := []string{"attach", "external-ip", pool, instance}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

// UnmapExternalIP unmaps an external ip from an instance. The address is
// unmapped using the ciao detach external-ip command. An error will be
// returned if the following environment variables are not set;
// CIAO_CLIENT_CERT_FILE, CIAO_CONTROLLER.
func UnmapExternalIP(ctx context.Context, tenant, address string) error {
	args := []string{"detach", "external-ip", address}
	_, err := RunCIAOCmd(ctx, tenant, args)
	return err
}

// DeleteExternalIPPool deletes an external-ip pool. The pool is deleted using
// the ciao delete pool command. An error will be returned if the following
// environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func DeleteExternalIPPool(ctx context.Context, tenant, name string) error {
	args := []string{"delete", "pool", name}
	_, err := RunCIAOCmdAsAdmin(ctx, tenant, args)
	return err
}

// ListExternalIPs returns detailed information about all the external ips //
// defined for the given tenant. The information is retrieved using the ciao
// list external-ips command. An error will be returned if the following //
// environment variables are not set; CIAO_ADMIN_CLIENT_CERT_FILE,
// CIAO_CONTROLLER.
func ListExternalIPs(ctx context.Context, tenant string) ([]*ExternalIP, error) {
	var externalIPs []*ExternalIP
	args := []string{"list", "external-ips", "-f", "{{tojson .}}"}
	err := RunCIAOCmdAsAdminJS(ctx, tenant, args, &externalIPs)
	if err != nil {
		return nil, err
	}

	return externalIPs, nil
}
