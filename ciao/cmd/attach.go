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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type volumeAttachFlags struct {
	instance   string
	mode       string
	mountpoint string
	volume     string
}

type externalipAttachFlags struct {
	instance string
	name     string
}

var volAttachFlags volumeAttachFlags
var externalipFlags externalipAttachFlags

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach various ciao objects",
	Long:  `Attach the objects below to the ciao cluster.`,
	Args:  cobra.MinimumNArgs(2),
}

var attachIpCmd = &cobra.Command{
	Use:  "external-ip",
	Long: `Attach an external IP from a given pool to an instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		if externalipFlags.instance == "" {
			fmt.Fprintf(os.Stderr, "Missing required --instance parameter")
			return
		}
		if externalipFlags.name == "" {
			fmt.Fprintf(os.Stderr, "Missing required --pool parameter")
			return
		}
		err := C.MapExternalIP(externalipFlags.name, externalipFlags.instance)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error mapping external IP: %s\n", err)
		}

		fmt.Printf("Requested external IP for: %s\n", externalipFlags.instance)
	},
}

var attachVolCmd = &cobra.Command{
	Use:  "volume",
	Long: `Attach a volume to an instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		if volAttachFlags.volume == "" {
			fmt.Fprintf(os.Stderr, "Missing required --volume parameter")
			return
		}

		if volAttachFlags.instance == "" {
			fmt.Fprintf(os.Stderr, "Missing required --instance parameter")
			return
		}

		err := C.AttachVolume(volAttachFlags.volume, volAttachFlags.instance, volAttachFlags.mountpoint, volAttachFlags.mode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error attaching volume: %s\n", err)
			return
		}

		fmt.Printf("Attached volume: %s\n", volAttachFlags.volume)
	},
}

func init() {
	attachCmd.AddCommand(attachIpCmd)
	attachCmd.AddCommand(attachVolCmd)

	RootCmd.AddCommand(attachCmd)

	attachVolCmd.Flags().StringVar(&volAttachFlags.instance, "instance", "", "Instance UUID")
	attachVolCmd.Flags().StringVar(&volAttachFlags.mode, "mode", "", "Access mode (default \"rw\")")
	attachVolCmd.Flags().StringVar(&volAttachFlags.mountpoint, "mountpoint", "", "Mount point (default \"/mnt\")")
	attachVolCmd.Flags().StringVar(&volAttachFlags.volume, "volume", "", "Volume UUID")

	attachIpCmd.Flags().StringVar(&externalipFlags.instance, "instance", "", "ID of the instance to map IP to")
	attachIpCmd.Flags().StringVar(&externalipFlags.name, "pool", "", "Name of the pool to map from")
}
