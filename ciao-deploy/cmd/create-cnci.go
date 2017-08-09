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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/01org/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

var anchorCertPath string
var caCertPath string

func createCNCI() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	sigCh := make(chan os.Signal, 1)
	go func() {
		<-sigCh
		cancelFunc()
	}()
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	err := deploy.CreateCNCIImage(ctx, anchorCertPath, caCertPath, imageCacheDirectory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating CNCI: %v\n", err)
		return 1
	}
	return 0
}

// createCNCICmd represents the create-cnci command
var createCNCICmd = &cobra.Command{
	Use:   "create-cnci",
	Short: "Populate cluster with CNCI image",
	Long: `Downloads the image needed for the CNCI, makes local customisations and
	 uploads to server`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(createCNCI())
	},
}

func init() {
	RootCmd.AddCommand(createCNCICmd)

	createCNCICmd.Flags().StringVar(&anchorCertPath, "anchor-cert-path", "", "Path to anchor certificate")
	createCNCICmd.Flags().StringVar(&caCertPath, "ca-cert-path", "", "Path to CA certificate")
	createCNCICmd.Flags().StringVar(&imageCacheDirectory, "image-cache-directory", deploy.DefaultImageCacheDir(), "Directory to use for caching of downloaded images")
}
