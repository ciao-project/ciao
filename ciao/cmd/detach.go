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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var detachCmd = &cobra.Command{
	Use:   "detach",
	Short: "Detach various ciao objects",
	Long:  `Detach the objects below from their given connections`,
}

var detachIPCmd = &cobra.Command{
	Use:  "external-ip IP",
	Long: `Detach an external IP`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.UnmapExternalIP(args[0]), "Error unmapping external IP")
	},
}

var detachVolCmd = &cobra.Command{
	Use:  "volume VOLUME",
	Long: `Detach a volume from an instance`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DetachVolume(args[0]), "Error detaching volume")
	},
}

func init() {
	detachCmd.AddCommand(detachIPCmd)
	detachCmd.AddCommand(detachVolCmd)

	rootCmd.AddCommand(detachCmd)
}
