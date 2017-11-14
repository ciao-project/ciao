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
	"github.com/pkg/errors"
)

func startStopInstance(instance string, stop bool) error {
	if C.TenantID == "" {
		return errors.New("Missing required -tenant-id parameter")
	}

	if instance == "" {
		return errors.New("Missing required -instance parameter")
	}

	if stop == true {
		err := C.StopInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error stopping instance")
		}
		fmt.Printf("Instance %s stopped\n", instance)
	} else {
		err := C.StartInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error starting instance")
		}
		fmt.Printf("Instance %s restarted\n", instance)
	}
	return nil
}

var restartCmd = &cobra.Command{
	Use:   "restart [UUID]",
	Short: "Restart a Ciao instance",
	Long:  `Restart a Ciao instance.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		instance := args[0]
		err := startStopInstance(instance, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restart instance %s: %s\n", instance, err)
		}
	},
}

func init() {
	RootCmd.AddCommand(restartCmd)
}
