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

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func checkEnv(vars []string) error {
	for _, k := range vars {
		if os.Getenv(k) == "" {
			return fmt.Errorf("%s is not defined", k)
		}
	}
	return nil
}

// RunCIAOCmd execs the ciao command with a set of arguments. The ciao
// process will be killed if the context is Done. An error will be returned if
// the following environment variables are not set; CIAO_CLIENT_CERT_FILE,
// CIAO_CONTROLLER. On success the data written to ciao on stdout will be
// returned.
func RunCIAOCmd(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_CLIENT_CERT_FILE", "CIAO_CONTROLLER"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	envCopy := os.Environ()

	if tenant != "" {
		envCopy = append(envCopy, fmt.Sprintf("CIAO_TENANT_ID=%s", tenant))
	}

	cmd := exec.CommandContext(ctx, "ciao", args...)
	cmd.Env = envCopy

	data, err := cmd.Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao %v : %v\n%s",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCmdJS is similar to RunCIAOCmd with the exception that the output
// of the ciao command is expected to be in json format.  The json is
// decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCmdJS(ctx context.Context, tenant string, args []string, jsdata interface{}) error {
	data, err := RunCIAOCmd(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return errors.Wrapf(err, "Cannot unmarshal data: %s", string(data))
	}

	return nil
}

// RunCIAOCmdAsAdmin execs the ciao command as the admin user with a set of
// provided arguments. The ciao process will be killed if the context is
// Done. An error will be returned if the following environment variables are
// not set; CIAO_ADMIN_CLIENT_CERT_FILE, CIAO_CONTROLLER. On success the data
// written to ciao on stdout will be returned.
func RunCIAOCmdAsAdmin(ctx context.Context, tenant string, args []string) ([]byte, error) {
	vars := []string{"CIAO_ADMIN_CLIENT_CERT_FILE", "CIAO_CONTROLLER"}
	if err := checkEnv(vars); err != nil {
		return nil, err
	}

	env := os.Environ()
	envCopy := make([]string, 0, len(env))
	for _, v := range env {
		if !strings.HasPrefix(v, "CIAO_CLIENT_CERT_FILE") {
			envCopy = append(envCopy, v)
		}
	}
	envCopy = append(envCopy, fmt.Sprintf("CIAO_CLIENT_CERT_FILE=%s",
		os.Getenv("CIAO_ADMIN_CLIENT_CERT_FILE")))

	if tenant != "" {
		envCopy = append(envCopy, fmt.Sprintf("CIAO_TENANT_ID=%s", tenant))
	}

	cmd := exec.CommandContext(ctx, "ciao", args...)
	cmd.Env = envCopy
	data, err := cmd.Output()
	if err != nil {
		var failureText string
		if err, ok := err.(*exec.ExitError); ok {
			failureText = string(err.Stderr)
		}
		return nil, fmt.Errorf("failed to launch ciao %v : %v\n%v",
			args, err, failureText)
	}

	return data, nil
}

// RunCIAOCmdAsAdminJS is similar to RunCIAOCmdAsAdmin with the exception that
// the output of the ciao command is expected to be in json format.  The
// json is decoded into the jsdata parameter which should be a pointer to a type
// that corresponds to the json output.
func RunCIAOCmdAsAdminJS(ctx context.Context, tenant string, args []string,
	jsdata interface{}) error {
	data, err := RunCIAOCmdAsAdmin(ctx, tenant, args)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, jsdata)
	if err != nil {
		return errors.Wrapf(err, "Cannot unmarshal data: %s", string(data))
	}

	return nil
}
