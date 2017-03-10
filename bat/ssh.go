//
// Copyright (c) 2016 Intel Corporation
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
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

//SSHClient contains details of the server to be connect
type SSHClient struct {
	ClientIP      string
	ClientPort    string
	ClientUser    string
	ClientPass    string
	ClientTimeout int64
}

func dialSSH(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	dialer := net.Dialer{
		Timeout: config.Timeout,
	}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to ssh server: %v", err)
	}
	c, chs, r, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create new SSH client connection : %v", err)
	}
	return ssh.NewClient(c, chs, r), nil
}

//createSSHClient creates a SSH session for the server specified
//
//Receive a struct with the details of the server that will be connected
//create a tcp session that will be use to send and receive data
//
//session sould be created succesfully and wait for commands
func createSSHClient(ctx context.Context, c SSHClient) (*ssh.Client, error) {
	//If no port is defined, it will be default to 22
	if c.ClientPort == "" {
		c.ClientPort = "22"
	}
	if c.ClientTimeout == 0 {
		c.ClientTimeout = 5
	}
	ClientFullAddress := fmt.Sprint(c.ClientIP + ":" + c.ClientPort)
	config := &ssh.ClientConfig{
		User: c.ClientUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.ClientPass),
		},
		Timeout: time.Duration(c.ClientTimeout) * time.Second,
	}
	//	Connection, err := ssh.Dial("tcp", ClientFullAddress, config)
	Connection, err := dialSSH(ctx, "tcp", ClientFullAddress, config)
	return Connection, err
}

//SSHSendCommand  send a command to a specified server
//
//Receive a struct with the details of the server that will be connected
//and the command that will be send to the server, set up the connection,
//send the command and returns server's response
//
//function returns output from the server to the specified command
func SSHSendCommand(ctx context.Context, client SSHClient, command string) (string, error) {
	//create a connection
	Connection, err := createSSHClient(ctx, client)
	if err != nil {
		return "", err
	}
	defer Connection.Close()
	//Create a session
	session, err := Connection.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	//send the command and get the result
	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run(command)
	if err != nil {
		return "", err
	}
	//return the result
	return b.String(), err
}
