// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"strconv"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update status of an object",
}

var updateQuotasCmd = &cobra.Command{
	Use:   "quota TENANT NAME VALUE",
	Short: "Update tenant quotas",
	Long:  "Updates the quota entry for the supplied tenant with the value or limit",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Updating quotas is restricted to privileged users")
		}

		tenant := args[0]
		name := args[1]
		value := args[2]

		var v int
		if value == "unlimited" {
			v = -1
		} else {
			var err error
			v, err = strconv.Atoi(value)
			if err != nil {
				return errors.Wrap(err, "Error converting to integer")
			}
		}

		quotas := []types.QuotaDetails{{
			Name:  name,
			Value: v,
		}}

		return errors.Wrap(c.UpdateQuotas(tenant, quotas), "Error updating quotas")
	},
}

var tenantUpdateCmd = &cobra.Command{
	Use:   "tenant ID",
	Short: "Update tenant configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Updating tenants is restricted to privileged users")
		}

		tenantID := args[0]

		// CIDR prefix size must be between 12 and 30 bits
		if tenantFlags.cidrPrefixSize != 0 && (tenantFlags.cidrPrefixSize > 30 || tenantFlags.cidrPrefixSize < 12) {
			return errors.New("Subnet prefix must be 12-30")
		}

		tuuid, err := uuid.Parse(tenantID)
		if err != nil {
			return errors.New("Tenant ID must be a UUID")
		}

		config := types.TenantConfig{
			Name:       tenantFlags.name,
			SubnetBits: tenantFlags.cidrPrefixSize,
		}
		config.Permissions.PrivilegedContainers = tenantFlags.createPrivilegedContainers

		return errors.Wrap(c.UpdateTenantConfig(tuuid.String(), config),
			"Error updating tenant config")
	},
}

func init() {
	updateCmd.AddCommand(updateQuotasCmd)
	updateCmd.AddCommand(tenantUpdateCmd)

	tenantUpdateCmd.Flags().IntVar(&tenantFlags.cidrPrefixSize, "cidr-prefix-size", 0, "Number of bits in network mask (12-30)")
	tenantUpdateCmd.Flags().BoolVar(&tenantFlags.createPrivilegedContainers, "create-privileged-containers", false, "Whether this tenant can create privileged containers")
	tenantUpdateCmd.Flags().StringVar(&tenantFlags.name, "name", "", "Tenant name")

	rootCmd.AddCommand(updateCmd)
}
