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

	"github.com/ciao-project/ciao/client"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var c client.Client

var template string

func render(cmd *cobra.Command, data interface{}) error {
	if template == "" && cmd.Annotations != nil {
		template = cmd.Annotations["default_template"]
	}

	if template == "" {
		template = "{{ htable (sliceof .) }}"
	}

	return errors.Wrap(tfortools.OutputToTemplate(os.Stdout, "", template, data, nil),
		"Error generating template output")
}

const (
	ciaoControllerEnv     = "CIAO_CONTROLLER"
	ciaoCACertFileEnv     = "CIAO_CA_CERT_FILE"
	ciaoClientCertFileEnv = "CIAO_CLIENT_CERT_FILE"
	ciaoTenantIDEnv       = "CIAO_TENANT_ID"
)

func getCiaoEnvVariables() {
	c.ControllerURL = os.Getenv(ciaoControllerEnv)
	c.CACertFile = os.Getenv(ciaoCACertFileEnv)
	c.ClientCertFile = os.Getenv(ciaoClientCertFileEnv)
	c.TenantID = os.Getenv(ciaoTenantIDEnv)
}

var rootCmd = &cobra.Command{
	Use: "ciao",
	Long: `
Command line interface for the Cloud Integrated Advanced Orchestrator (CIAO).

The CIAO CLI sends HTTPS requests to the CIAO controller enabling one to control a CIAO cluster.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	getCiaoEnvVariables()
	if err := c.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init the CLI: %s\n", err)
		os.Exit(1)
	}

	rootCmd.PersistentFlags().StringVarP(&template, "template", "f", "", "Template used to format output")
}
