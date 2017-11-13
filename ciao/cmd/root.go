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

	"github.com/ciao-project/ciao/ciao-sdk"
	"github.com/ciao-project/ciao/client"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var C client.Client

var CommandFlags = new(sdk.CommandOpts)

var cfgFile string

var (
	tenantID       = new(string)
	controllerURL  = new(string)
	ciaoPort       = new(int)
	caCertFile     = new(string)
	clientCertFile = new(string)
)

const (
	ciaoControllerEnv     = "CIAO_CONTROLLER"
	ciaoCACertFileEnv     = "CIAO_CA_CERT_FILE"
	ciaoClientCertFileEnv = "CIAO_CLIENT_CERT_FILE"
)

func getCiaoEnvVariables() {
	controller := os.Getenv(ciaoControllerEnv)
	ca := os.Getenv(ciaoCACertFileEnv)
	clientCert := os.Getenv(ciaoClientCertFileEnv)

	client.Infof("Ciao environment variables:\n")
	client.Infof("\t%s:%s\n", ciaoControllerEnv, controller)
	client.Infof("\t%s:%s\n", ciaoCACertFileEnv, ca)
	client.Infof("\t%s:%s\n", ciaoClientCertFileEnv, clientCert)

	C.ControllerURL = controller
	C.CACertFile = ca
	C.ClientCertFile = clientCert

	if *controllerURL != "" {
		C.ControllerURL = *controllerURL
	}

	if *caCertFile != "" {
		C.CACertFile = *caCertFile
	}

	if *clientCertFile != "" {
		C.ClientCertFile = *clientCertFile
	}

	if *tenantID != "" {
		C.TenantID = *tenantID
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: "ciao",
	Long: `
Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO).

The ciao cli sends HTTPS requests to the Ciao controller compute API endpoints,
enabling one to get information and control a Ciao cluster.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	getCiaoEnvVariables()
	C.Init()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".ciao" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ciao")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
