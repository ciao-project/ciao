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

package ssntp

import (
	"bytes"
	"encoding/asn1"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"
)

type ssntpEchoServer struct {
	ssntp Server
	t     *testing.T

	roleConnectChannel    chan string
	roleDisconnectChannel chan string
	majorChannel          chan struct{}
}

func (server *ssntpEchoServer) ConnectNotify(uuid string, role uint32) {
	if server.roleConnectChannel != nil {
		sRole := (Role)(role)
		server.roleConnectChannel <- sRole.String()
	}
}

func (server *ssntpEchoServer) DisconnectNotify(uuid string, role uint32) {
	if server.roleDisconnectChannel != nil {
		sRole := (Role)(role)
		server.roleDisconnectChannel <- sRole.String()
	}
}

func (server *ssntpEchoServer) StatusNotify(uuid string, status Status, frame *Frame) {
	server.ssntp.SendStatus(uuid, status, frame.Payload)
}

func (server *ssntpEchoServer) CommandNotify(uuid string, command Command, frame *Frame) {
	if server.majorChannel != nil {
		if frame.major() == major {
			close(server.majorChannel)
		}
	}

	server.ssntp.SendCommand(uuid, command, frame.Payload)
}

func (server *ssntpEchoServer) EventNotify(uuid string, event Event, frame *Frame) {
	server.ssntp.SendEvent(uuid, event, frame.Payload)
}

func (server *ssntpEchoServer) ErrorNotify(uuid string, error Error, frame *Frame) {
	server.ssntp.SendError(uuid, error, frame.Payload)
}

type ssntpEchoFwderServer struct {
	ssntp Server
	t     *testing.T
}

func (server *ssntpEchoFwderServer) ConnectNotify(uuid string, role uint32) {
}

func (server *ssntpEchoFwderServer) DisconnectNotify(uuid string, role uint32) {
}

func (server *ssntpEchoFwderServer) StatusNotify(uuid string, status Status, frame *Frame) {
}

func (server *ssntpEchoFwderServer) CommandNotify(uuid string, command Command, frame *Frame) {
}

func (server *ssntpEchoFwderServer) EventNotify(uuid string, event Event, frame *Frame) {
}

func (server *ssntpEchoFwderServer) ErrorNotify(uuid string, error Error, frame *Frame) {
}

func (server *ssntpEchoFwderServer) CommandForward(uuid string, command Command, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(uuid)

	return
}

func (server *ssntpEchoFwderServer) EventForward(uuid string, event Event, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(uuid)

	return
}

func (server *ssntpEchoFwderServer) StatusForward(uuid string, status Status, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(uuid)

	return
}

func (server *ssntpEchoFwderServer) ErrorForward(uuid string, error Error, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(uuid)

	return
}

type ssntpServer struct {
	ssntp Server
	t     *testing.T
}

func (server *ssntpServer) ConnectNotify(uuid string, role uint32) {
}

func (server *ssntpServer) DisconnectNotify(uuid string, role uint32) {
}

func (server *ssntpServer) StatusNotify(uuid string, status Status, frame *Frame) {
}

func (server *ssntpServer) CommandNotify(uuid string, command Command, frame *Frame) {
}

func (server *ssntpServer) EventNotify(uuid string, event Event, frame *Frame) {
}

func (server *ssntpServer) ErrorNotify(uuid string, error Error, frame *Frame) {
}

func (server *ssntpServer) CommandForward(uuid string, command Command, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(controllerUUID)

	return
}

func (server *ssntpServer) EventForward(uuid string, event Event, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(controllerUUID)

	return
}

func (server *ssntpServer) StatusForward(uuid string, status Status, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(controllerUUID)

	return
}

func (server *ssntpServer) ErrorForward(uuid string, error Error, frame *Frame) (dest ForwardDestination) {
	dest.AddRecipient(controllerUUID)

	return
}

type ssntpClient struct {
	ssntp        Client
	t            *testing.T
	payload      []byte
	disconnected chan struct{}
	connected    chan struct{}
	typeChannel  chan string
	cmdChannel   chan string
	staChannel   chan string
	evtChannel   chan string
	errChannel   chan string

	cmdTracedChannel   chan string
	cmdDurationChannel chan time.Duration
	cmdDumpChannel     chan struct{}
	staTracedChannel   chan string
	evtTracedChannel   chan string
	errTracedChannel   chan string
}

func (client *ssntpClient) ConnectNotify() {
	if client.connected != nil {
		close(client.connected)
	}
}

func (client *ssntpClient) DisconnectNotify() {
	if client.disconnected != nil {
		close(client.disconnected)
	}
}

func (client *ssntpClient) StatusNotify(status Status, frame *Frame) {
	if client.typeChannel != nil {
		client.typeChannel <- STATUS.String()
	}

	if client.staChannel != nil && bytes.Equal(frame.Payload, client.payload) == true {
		client.staChannel <- status.String()
	}
}

func (client *ssntpClient) CommandNotify(command Command, frame *Frame) {
	if client.typeChannel != nil {
		client.typeChannel <- COMMAND.String()
	}

	if client.cmdChannel != nil && bytes.Equal(frame.Payload, client.payload) == true {
		client.cmdChannel <- command.String()
	}

	if client.cmdDumpChannel != nil {
		trace, err := frame.DumpTrace()
		if err == nil && trace.Type == COMMAND.String() {
			close(client.cmdDumpChannel)
		}
	}

	if client.cmdTracedChannel != nil {
		if frame.Trace.Label != nil {
			client.cmdTracedChannel <- string(frame.Trace.Label)
		} else if frame.PathTrace() == true {
			client.cmdTracedChannel <- string(frame.Trace.PathLength)
			duration, _ := frame.Duration()
			client.cmdDurationChannel <- duration
		} else {
			close(client.cmdTracedChannel)
		}
	} else {
		if client.cmdDurationChannel != nil {
			_, err := frame.Duration()
			if err != nil {
				client.cmdDurationChannel <- 0
			}
		}
	}
}

func (client *ssntpClient) EventNotify(event Event, frame *Frame) {
	if client.typeChannel != nil {
		client.typeChannel <- EVENT.String()
	}

	if client.evtChannel != nil && bytes.Equal(frame.Payload, client.payload) == true {
		client.evtChannel <- event.String()
	}
}

func (client *ssntpClient) ErrorNotify(error Error, frame *Frame) {
	if client.typeChannel != nil {
		client.typeChannel <- ERROR.String()
	}

	if client.errChannel != nil && bytes.Equal(frame.Payload, client.payload) == true {
		client.errChannel <- error.String()
	}
}

// Test client UUID generation code
//
// Test that two consecutive SSNTP clients get the same UUID.
// This test verifies that the client UUID permanent storage
// code path works fine.
//
// Test is expected to pass.
func TestUUID(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client1, client2 ssntpClient

	server.t = t
	client1.t = t
	client2.t = t
	serverConfig.Transport = *transport
	serverConfig.Role = SERVER
	clientConfig.Transport = *transport
	clientConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)

	time.Sleep(500 * time.Millisecond)
	client1.ssntp.Dial(&clientConfig, &client1)
	client1.ssntp.Close()

	err := client2.ssntp.Dial(&clientConfig, &client2)
	if err != nil {
		t.Fatalf("Failed to connect %s", err)
	}

	if client1.ssntp.UUID() != client2.ssntp.UUID() {
		client2.ssntp.Close()
		t.Fatalf("Wrong client UUID %s vs %s", client1.ssntp.UUID(), client2.ssntp.UUID())
	}

	client2.ssntp.Close()
	server.ssntp.Stop()
}

func testGetOIDFromRole(t *testing.T, role uint32, expected asn1.ObjectIdentifier) {
	oid, err := getOIDFromRole(role)
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	if !oid.Equal(expected) {
		t.Fatalf("OID mismatch for %d: %v vs %v\n", role, oid, expected)
	}
}

func testGetRoleFromOID(t *testing.T, oid asn1.ObjectIdentifier, expected uint32) {
	role, err := getRoleFromOID(oid)
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	if role != expected {
		t.Fatalf("Role mismatch for %v: %d vs %d\n", oid, role, expected)
	}
}

// Test SSNTP Agent OID match
//
// Test that we get the right OID for the AGENT role.
//
// Test is expected to pass.
func TestGetOIDFromAgent(t *testing.T) {
	testGetOIDFromRole(t, AGENT, RoleAgentOID)
}

// Test SSNTP Scheduler OID match
//
// Test that we get the right OID for the SCHEDULER role.
//
// Test is expected to pass.
func TestGetOIDFromSchedulerRole(t *testing.T) {
	testGetOIDFromRole(t, SCHEDULER, RoleSchedulerOID)
}

// Test SSNTP Controller OID match
//
// Test that we get the right OID for the Controller role.
//
// Test is expected to pass.
func TestGetOIDFromControllerRole(t *testing.T) {
	testGetOIDFromRole(t, Controller, RoleControllerOID)
}

// Test SSNTP NetAgent OID match
//
// Test that we get the right OID for the NETAGENT role.
//
// Test is expected to pass.
func TestGetOIDFromNetAgentRole(t *testing.T) {
	testGetOIDFromRole(t, NETAGENT, RoleNetAgentOID)
}

// Test SSNTP Server OID match
//
// Test that we get the right OID for the SERVER role.
//
// Test is expected to pass.
func TestGetOIDFromServerRole(t *testing.T) {
	testGetOIDFromRole(t, SERVER, RoleServerOID)
}

// Test SSNTP CNCI Agent OID match
//
// Test that we get the right OID for the CNCIAGENT role.
//
// Test is expected to pass.
func TestGetOIDFromCNCIAgentRole(t *testing.T) {
	testGetOIDFromRole(t, CNCIAGENT, RoleCNCIAgentOID)
}

// Test SSNTP OID match for an invalid role
//
// Test that we do not get a valid OID for an invalid role.
//
// Test is expected to pass.
func TestGetOIDFromInvalidRole(t *testing.T) {
	_, err := getOIDFromRole(0xffff)
	if err == nil {
		t.Fatalf("Got OID for an invalid role\n")
	}
}

// Test SSNTP CNCI Agent role match
//
// Test that we get the right role for the Agent OID.
//
// Test is expected to pass.
func TestGetRoleFromAgentOID(t *testing.T) {
	testGetRoleFromOID(t, RoleAgentOID, AGENT)
}

// Test SSNTP CNCI Scheduler role match
//
// Test that we get the right role for the Scheduler OID.
//
// Test is expected to pass.
func TestGetRoleFromSchedulerOID(t *testing.T) {
	testGetRoleFromOID(t, RoleSchedulerOID, SCHEDULER)
}

// Test SSNTP CNCI Controller role match
//
// Test that we get the right role for the Controller OID.
//
// Test is expected to pass.
func TestGetRoleFromControllerOID(t *testing.T) {
	testGetRoleFromOID(t, RoleControllerOID, Controller)
}

// Test SSNTP CNCI Server role match
//
// Test that we get the right role for the Server OID.
//
// Test is expected to pass.
func TestGetRoleFromServerOID(t *testing.T) {
	testGetRoleFromOID(t, RoleServerOID, SERVER)
}

// Test SSNTP CNCI Net Agent role match
//
// Test that we get the right role for the Net Agent OID.
//
// Test is expected to pass.
func TestGetRoleFromNetAgentOID(t *testing.T) {
	testGetRoleFromOID(t, RoleNetAgentOID, NETAGENT)
}

// Test SSNTP CNCI Agent role match
//
// Test that we get the right role for the CNCI Agent OID.
//
// Test is expected to pass.
func TestGetRoleFromCNCIAgentOID(t *testing.T) {
	testGetRoleFromOID(t, RoleCNCIAgentOID, CNCIAGENT)
}

// Test SSNTP CNCI Agent role match for an invalid OID
//
// Test that we get the UNKNOWN role for an invalid OID.
//
// Test is expected to pass.
func TestGetRoleFromInvalidOID(t *testing.T) {
	testGetRoleFromOID(t, asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 0, 0, 0}, (uint32)(UNKNOWN))
}

// Test SSNTP client connection
//
// Test that an SSNTP client can connect to an SSNTP server.
//
// Test is expected to pass.
func TestConnect(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)

	client.ssntp.Close()
	server.ssntp.Stop()

	if err != nil {
		t.Fatalf("Failed to connect")
	}
}

func testConnectRole(t *testing.T, role Role) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	server.roleConnectChannel = make(chan string)
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport
	clientConfig.Role = (uint32)(role)

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	select {
	case clientRole := <-server.roleConnectChannel:
		if clientRole != role.String() {
			t.Fatalf("Wrong role")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the connection notification")
	}

	client.ssntp.Close()
	server.ssntp.Stop()
}

func testDisconnectRole(t *testing.T, role Role) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	server.roleDisconnectChannel = make(chan string)
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport
	clientConfig.Role = (uint32)(role)

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.ssntp.Close()

	select {
	case clientRole := <-server.roleDisconnectChannel:
		if clientRole != role.String() {
			t.Fatalf("Wrong role")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the disconnection notification")
	}

	server.ssntp.Stop()
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as a SERVER can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleServer(t *testing.T) {
	testConnectRole(t, SERVER)
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as a Controller can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleController(t *testing.T) {
	testConnectRole(t, Controller)
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as an AGENT can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleAgent(t *testing.T) {
	testConnectRole(t, AGENT)
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as a SCHEDULER can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleScheduler(t *testing.T) {
	testConnectRole(t, SCHEDULER)
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as a NETAGENT can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleNetAgent(t *testing.T) {
	testConnectRole(t, NETAGENT)
}

// Test the SSNTP client role from the server connection.
//
// Test that a SSNTP client acting as a CNCIAGENT can
// connect to a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestConnectRoleCNCIAgent(t *testing.T) {
	testConnectRole(t, CNCIAGENT)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as a SERVER can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleServer(t *testing.T) {
	testDisconnectRole(t, SERVER)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as a Controller can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleController(t *testing.T) {
	testDisconnectRole(t, Controller)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as an AGENT can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleAgent(t *testing.T) {
	testDisconnectRole(t, AGENT)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as a SCHEDULER can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleScheduler(t *testing.T) {
	testDisconnectRole(t, SCHEDULER)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as a NETAGENT can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleNetAgent(t *testing.T) {
	testDisconnectRole(t, NETAGENT)
}

// Test the SSNTP client role from the server disconnection.
//
// Test that a SSNTP client acting as a CNCIAGENT can
// disconnect from a SSNTP server, and that the server sees
// the right role.
//
// Test is expected to pass.
func TestDisconnectRoleCNCIAgent(t *testing.T) {
	testDisconnectRole(t, CNCIAGENT)
}

func TestMajor(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	server.majorChannel = make(chan struct{})
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendCommand(START, client.payload)

	select {
	case <-server.majorChannel:
		break
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the major frame")
	}

	client.ssntp.Close()
	server.ssntp.Stop()
}

/* Mark D. Ryan FTW ! */
func getCertPaths(tmpDir, caCert, serverCert, clientCert string) (string, string, string) {
	var caPath, serverPath, clientPath string

	caPath = path.Join(tmpDir, "CACert")
	serverPath = path.Join(tmpDir, "ServerCert")
	clientPath = path.Join(tmpDir, "ClientCert")

	for _, s := range []struct{ path, data string }{{caPath, caCert}, {serverPath, serverCert}, {clientPath, clientCert}} {
		err := ioutil.WriteFile(s.path, []byte(s.data), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create certfile %s %v\n", s.path, err)
			os.Exit(1)
		}
	}

	return caPath, serverPath, clientPath
}

func validRoles(serverRole, clientRole Role) bool {
	if serverRole == SCHEDULER && clientRole == AGENT {
		return true
	}

	return false
}

func testConnectVerifyCertificate(t *testing.T, serverRole, clientRole Role) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	tmpDir, err := ioutil.TempDir("", "ssntp-test-certs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create temporary Dir %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	CACert, serverCert, clientCert := getCertPaths(tmpDir, testCACertScheduler, testCertScheduler, testCertAgent)

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.RoleVerification = true
	serverConfig.CAcert = CACert
	serverConfig.Cert = serverCert
	serverConfig.Role = (uint32)(serverRole)

	client.t = t
	clientConfig.Transport = *transport
	clientConfig.RoleVerification = true
	clientConfig.CAcert = CACert
	clientConfig.Cert = clientCert
	clientConfig.Role = (uint32)(clientRole)

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err = client.ssntp.Dial(&clientConfig, &client)

	client.ssntp.Close()
	server.ssntp.Stop()

	if validRoles(serverRole, clientRole) && err != nil {
		t.Fatalf("Failed to connect")
	}

	if !validRoles(serverRole, clientRole) && err == nil {
		t.Fatalf("Wrong certificate, connection should not be allowed")
	}
}

// Test that an SSNTP verified link can be established.
//
// Test that an SSNTP client can connect to an SSNTP server
// when both are using SSNTP specific certificates.
//
// Test is expected to pass.
func TestConnectVerifyCertificatePositive(t *testing.T) {
	testConnectVerifyCertificate(t, SCHEDULER, AGENT)
}

// Test that an SSNTP verified link with the wrong client
// certificate should not be established.
//
// Test that an SSNTP client can not connect to an SSNTP server
// when both are using SSNTP specific certificates and the client
// has not defined the right role.
//
// Test is expected to pass.
func TestConnectVerifyClientCertificateNegative(t *testing.T) {
	testConnectVerifyCertificate(t, SCHEDULER, Controller)
}

// Test that an SSNTP verified link with the wrong server
// certificate should not be established.
//
// Test that an SSNTP client can not connect to an SSNTP server
// when both are using SSNTP specific certificates and the client
// has not defined the right role.
//
// Test is expected to pass.
func TestConnectVerifyServerCertificateNegative(t *testing.T) {
	testConnectVerifyCertificate(t, SERVER, AGENT)
}

// Test SSNTP client connection to an alternative port
//
// Test that an SSNTP client can connect to an SSNTP server
// listening to a non standard SSNTP port (i.e. different than
// 8888).
//
// Test is expected to pass.
func TestConnectPort(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	serverConfig.Transport = *transport
	serverConfig.Port = 9999
	clientConfig.Transport = *transport
	clientConfig.Port = 9999

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}
	client.ssntp.Close()
	server.ssntp.Stop()
}

// Test SSNTP client connection closure before Dial.
//
// Test that an SSNTP client can close itself before Dialing
// into the server. We verifiy that the subsequent Dial() call
// should fail.
//
// Test is expected to pass.
func TestClientCloseBeforeDial(t *testing.T) {
	var clientConfig Config
	var client ssntpClient

	client.t = t
	clientConfig.Transport = *transport

	client.ssntp.Close()
	err := client.ssntp.Dial(&clientConfig, &client)
	if err == nil {
		t.Fatalf("Initiated connection while closed")
	}
}

// Test SSNTP client connection closure after Dial.
//
// Test that an SSNTP client can close itself after Dialing
// into the server.
//
// Test is expected to pass.
func TestClientCloseAfterDial(t *testing.T) {
	var clientConfig Config
	var client ssntpClient

	client.t = t
	clientConfig.Transport = *transport

	go client.ssntp.Dial(&clientConfig, &client)
	time.Sleep(1000 * time.Millisecond)
	client.ssntp.Close()
	time.Sleep(5000 * time.Millisecond)
}

// Test SSNTP client reconnection to a server.
//
// Test that an SSNTP client eventually reconnects to
// a SSNTP server that restarts.
//
// Test is expected to pass.
func TestClientReconnect(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	client.connected = make(chan struct{})
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	select {
	case <-client.connected:
		break
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the connection notification")
	}

	client.disconnected = make(chan struct{})

	server.ssntp.Stop()

	select {
	case <-client.disconnected:
		break
	case <-time.After(3 * time.Second):
		t.Fatalf("Did not receive the disconnection notification")
	}

	client.connected = make(chan struct{})
	go server.ssntp.Serve(&serverConfig, &server)

	select {
	case <-client.connected:
		break
	case <-time.After(10 * time.Second):
		t.Fatalf("Did not receive the disconnection notification")
	}

	client.ssntp.Close()
	server.ssntp.Stop()
}

// Test SSNTP server Stop()
//
// Test that an SSNTP client properly receives its disconnection
// notification when its server stops.
//
// Test is expected to pass.
func TestServerStop(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	client.connected = make(chan struct{})
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	select {
	case <-client.connected:
		break
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the connection notification")
	}

	client.disconnected = make(chan struct{})

	server.ssntp.Stop()

	select {
	case <-client.disconnected:
		break
	case <-time.After(3 * time.Second):
		t.Fatalf("Did not receive the disconnection notification")
	}

	client.ssntp.Close()
	time.Sleep(500 * time.Millisecond)
}

// Test SSNTP Command frame
//
// Test that an SSNTP client can send a Command frame to an echo
// server and then receives it back consistently.
//
// Test is expected to pass.
func TestCommand(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdChannel = make(chan string)
	client.typeChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendCommand(START, client.payload)

	defer func() {
		client.ssntp.Close()
		server.ssntp.Stop()
	}()

	select {
	case frameType := <-client.typeChannel:
		if frameType != COMMAND.String() {
			t.Fatalf("Did not receive the right frame type")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the command notification")
	}

	select {
	case check := <-client.cmdChannel:
		if check != START.String() {
			t.Fatalf("Did not receive the right payload")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the command notification")
	}
}

// Test SSNTP Command traced frame label
//
// Test that an SSNTP client can send a traced Command frame to an echo
// server and then receives it back consistently.
// We test that the label is received back as expected.
//
// Test is expected to pass.
func TestTracedLabelCommand(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoFwderServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdTracedChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        START,
			CommandForward: &server,
		},
	}

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	clientLabel := "LabelClient"
	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendTracedCommand(START, client.payload,
		&TraceConfig{
			Label: []byte(clientLabel),
		},
	)

	check := <-client.cmdTracedChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	if check != clientLabel {
		t.Fatalf("Did not receive the right payload")
	}
}

// Test SSNTP Command traced frame networking path
//
// Test that an SSNTP client can send a traced Command frame to an echo
// server and then receives it back consistently.
// We test that the number of networking nodes received as part of the
// echo server reply is the right one.
//
// Test is expected to pass.
func TestTracedPathCommand(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoFwderServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdTracedChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        START,
			CommandForward: &server,
		},

		{
			Operand:       READY,
			StatusForward: &server,
		},
	}

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendTracedCommand(START, client.payload,
		&TraceConfig{
			PathTrace: true,
		},
	)

	check := <-client.cmdTracedChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	/* We should get 3 nodes */
	if check != string(3) {
		t.Fatalf("Did not receive the right payload %s", check)
	}
}

// Test SSNTP Command traced frame dump
//
// Test that an SSNTP client can send a traced Command frame to an echo
// server, receives it back consistently and dump it.
//
// Test is expected to pass.
func TestDumpTracedCommand(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoFwderServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdDumpChannel = make(chan struct{})
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        START,
			CommandForward: &server,
		},

		{
			Operand:       READY,
			StatusForward: &server,
		},
	}

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendTracedCommand(START, client.payload,
		&TraceConfig{
			PathTrace: true,
		},
	)

	defer func() {
		client.ssntp.Close()
		server.ssntp.Stop()
	}()

	select {
	case <-client.cmdDumpChannel:
		break
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the dump notification")
	}
}

// Test SSNTP Command traced frame duration
//
// Test that an SSNTP client can send a traced Command frame to an echo
// server and then receives it back consistently.
// We test that the frame duration is not zero.
//
// Test is expected to pass.
func TestCommandDuration(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoFwderServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdTracedChannel = make(chan string)
	client.cmdDurationChannel = make(chan time.Duration)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        START,
			CommandForward: &server,
		},

		{
			Operand:       READY,
			StatusForward: &server,
		},
	}

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendTracedCommand(START, client.payload,
		&TraceConfig{
			PathTrace: true,
		},
	)

	check := <-client.cmdTracedChannel
	duration := <-client.cmdDurationChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	/* We should get 3 nodes */
	if check != string(3) {
		t.Fatalf("Wrong number of nodes %s", check)
	}

	/* We should get a non zero duration */
	if duration == 0 {
		t.Fatalf("Zero duration")
	}
}

// Test the lack of duration on a non traced Command frame
//
// Test that we can not compute a duration on a non traced
// frame that comes back to the client.
//
// Test is expected to pass.
func TestCommandNoDuration(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdChannel = make(chan string)
	client.cmdDurationChannel = make(chan time.Duration)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendCommand(RESTART, client.payload)

	defer func() {
		client.ssntp.Close()
		server.ssntp.Stop()
	}()

	select {
	case check := <-client.cmdChannel:
		if check != RESTART.String() {
			t.Fatalf("Did not receive the right payload")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the command notification")
	}

	select {
	case duration := <-client.cmdDurationChannel:
		if duration != 0 {
			t.Fatalf("Should not receive a duration")
		}
	case <-time.After(time.Second):
		t.Fatalf("Did not receive the duration notification")
	}
}

// Test sending consecutive frames
//
// Test that an SSNTP client can send several SSNTP frames to an echo
// sever and then receives it back consistently and in order.
//
// Test is expected to pass.
func TestConsecutiveFrames(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoFwderServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.cmdChannel = make(chan string)
	client.staChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        DELETE,
			CommandForward: &server,
		},
		{
			Operand:       READY,
			StatusForward: &server,
		},
	}

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'Y', 'A', 'M', 'L'}
	client.ssntp.SendStatus(READY, client.payload)

	client.payload = []byte{'D', 'E', 'L', 'E', 'T', 'E'}
	client.ssntp.SendCommand(DELETE, client.payload)

	check := <-client.cmdChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	if check != DELETE.String() {
		t.Fatalf("Did not receive the right payload")
	}
}

// Test SSNTP Status frame
//
// Test that an SSNTP client can send a Status frame to an echo
// server and then receives it back consistently.
//
// Test is expected to pass.
func TestStatus(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.typeChannel = make(chan string)
	client.staChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'R', 'E', 'A', 'D', 'Y'}
	client.ssntp.SendStatus(READY, client.payload)

	frameType := <-client.typeChannel
	if frameType != STATUS.String() {
		t.Fatalf("Did not receive the right frame type")
	}

	check := <-client.staChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	if check != READY.String() {
		t.Fatalf("Did not receive the right payload")
	}
}

// Test SSNTP Event frame
//
// Test that an SSNTP client can send an Event frame to an echo
// server and then receives it back consistently.
//
// Test is expected to pass.
func TestEvent(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.typeChannel = make(chan string)
	client.evtChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'T', 'E', 'N', 'A', 'N', 'T'}
	client.ssntp.SendEvent(TenantAdded, client.payload)

	frameType := <-client.typeChannel
	if frameType != EVENT.String() {
		t.Fatalf("Did not receive the right frame type")
	}

	check := <-client.evtChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	if check != TenantAdded.String() {
		t.Fatalf("Did not receive the right payload")
	}
}

// Test SSNTP Error frame
//
// Test that an SSNTP client can send an Error frame to an echo
// server and then receives it back consistently.
//
// Test is expected to pass.
func TestError(t *testing.T) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpEchoServer
	var client ssntpClient

	server.t = t
	client.t = t
	client.typeChannel = make(chan string)
	client.errChannel = make(chan string)
	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := client.ssntp.Dial(&clientConfig, &client)
	if err != nil {
		t.Fatalf("Failed to connect")
	}

	client.payload = []byte{'E', 'R', 'R', 'O', 'R'}
	client.ssntp.SendError(InvalidFrameType, client.payload)

	frameType := <-client.typeChannel
	if frameType != ERROR.String() {
		t.Fatalf("Did not receive the right frame type")
	}

	check := <-client.errChannel

	client.ssntp.Close()
	server.ssntp.Stop()

	if check != InvalidFrameType.String() {
		t.Fatalf("Did not receive the right payload")
	}
}

// Test SSNTP Command forwarding
//
// Start an SSNTP server with a set of forwarding rules, an SSNTP
// agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as specified by the server forwarding rules.
//
// Test is expected to pass.
func TestCmdFwd(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	command := STOP

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand: command,
			Dest:    Controller,
		},
	}

	controller.t = t
	controller.cmdChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'S', 'T', 'A', 'T', 'S'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendCommand(command, agent.payload)

	check := <-controller.cmdChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != command.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

const controllerUUID = "3390740c-dce9-48d6-b83a-a717417072ce"

// Test SSNTP Command forwarder implementation
//
// Start an SSNTP server with a set of forwarding rules implemented
// by a command forwarder, an SSNTP agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as implemented by the server forwarder.
//
// Test is expected to pass.
func TestCmdFwder(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	command := EVACUATE

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:        command,
			CommandForward: &server,
		},
	}

	controller.t = t
	controller.cmdChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller
	controllerConfig.UUID = controllerUUID

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'E', 'V', 'A', 'C', 'U', 'A', 'T', 'E'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendCommand(command, agent.payload)

	check := <-controller.cmdChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != command.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Event forwarding
//
// Start an SSNTP server with a set of forwarding rules, an SSNTP
// agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as specified by the server forwarding rules.
//
// Test is expected to pass.
func TestEventFwd(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	event := TenantAdded

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand: event,
			Dest:    Controller,
		},
	}

	controller.t = t
	controller.evtChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'T', 'E', 'N', 'A', 'N', 'T'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendEvent(event, agent.payload)

	check := <-controller.evtChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != event.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Event forwarder implementation
//
// Start an SSNTP server with a set of forwarding rules implemented
// by an event forwarder, an SSNTP agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as implemented by the server forwarder.
//
// Test is expected to pass.
func TestEventFwder(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	event := TenantRemoved

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:      event,
			EventForward: &server,
		},
	}

	controller.t = t
	controller.evtChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller
	controllerConfig.UUID = controllerUUID

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'T', 'E', 'N', 'A', 'N', 'T'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendEvent(event, agent.payload)

	check := <-controller.evtChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != event.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Error forwarding
//
// Start an SSNTP server with a set of forwarding rules, an SSNTP
// agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as specified by the server forwarding rules.
//
// Test is expected to pass.
func TestErrorFwd(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	error := StartFailure

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand: error,
			Dest:    Controller,
		},
	}

	controller.t = t
	controller.errChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'F', 'A', 'I', 'L', 'E', 'D'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendError(error, agent.payload)

	check := <-controller.errChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != error.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Error forwarder implementation
//
// Start an SSNTP server with a set of forwarding rules implemented
// by an error forwarder, an SSNTP agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as implemented by the server forwarder.
//
// Test is expected to pass.
func TestErrorFwder(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	error := StopFailure

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:      error,
			ErrorForward: &server,
		},
	}

	controller.t = t
	controller.errChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller
	controllerConfig.UUID = controllerUUID

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'F', 'A', 'I', 'L', 'E', 'D'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendError(error, agent.payload)

	check := <-controller.errChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != error.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Command forwarding
//
// Start an SSNTP server with a set of forwarding rules, an SSNTP
// agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as specified by the server forwarding rules.
//
// Test is expected to pass.
func TestStatusFwd(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	status := FULL

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand: status,
			Dest:    Controller,
		},
	}

	controller.t = t
	controller.staChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'F', 'U', 'L', 'L'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendStatus(status, agent.payload)

	check := <-controller.staChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != status.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

// Test SSNTP Status forwarder implementation
//
// Start an SSNTP server with a set of forwarding rules implemented
// by a status forwarder, an SSNTP agent and an SSNTP Controller.
// Then verify that the Controller receives the right frames sent by the agent,
// as implemented by the server forwarder.
//
// Test is expected to pass.
func TestStatusFwder(t *testing.T) {
	var serverConfig Config
	var controllerConfig, agentConfig Config
	var server ssntpServer
	var controller, agent ssntpClient
	status := OFFLINE

	server.t = t
	serverConfig.Transport = *transport
	serverConfig.ForwardRules = []FrameForwardRule{
		{
			Operand:       status,
			StatusForward: &server,
		},
	}

	controller.t = t
	controller.staChannel = make(chan string)
	controllerConfig.Transport = *transport
	controllerConfig.Role = Controller
	controllerConfig.UUID = controllerUUID

	agent.t = t
	agentConfig.Transport = *transport
	agentConfig.Role = AGENT

	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	err := controller.ssntp.Dial(&controllerConfig, &controller)
	if err != nil {
		t.Fatalf("Controller failed to connect")
	}

	err = agent.ssntp.Dial(&agentConfig, &agent)
	if err != nil {
		t.Fatalf("Agent failed to connect")
	}

	payload := []byte{'O', 'F', 'F', 'L', 'I', 'N', 'E'}
	controller.payload = payload
	agent.payload = payload
	agent.ssntp.SendStatus(status, agent.payload)

	check := <-controller.staChannel

	agent.ssntp.Close()
	controller.ssntp.Close()
	server.ssntp.Stop()

	if check != status.String() {
		t.Fatalf("Did not receive the forwarded STATS")
	}
}

var (
	transport   = flag.String("transport", "tcp", "SSNTP transport, must be tcp or unix")
	clients     = flag.Int("clients", 100, "Number of clients to create for benchmarking")
	delay       = flag.Int("delay", 10, "Milliseconds between each client transmission")
	frames      = flag.Int("frames", 1000, "Number of frames per client to send")
	payloadSize = flag.Int("payload", 1<<11, "Frames payload size")
)

func TestMain(m *testing.M) {
	flag.Parse()

	if *transport != "tcp" && *transport != "unix" {
		*transport = "tcp"
	}

	os.Exit(m.Run())
}

type ssntpNullServer struct {
	ssntp Server
	b     *testing.B
	nCmds int
	wg    sync.WaitGroup
	done  chan struct{}
}

func (server *ssntpNullServer) ConnectNotify(uuid string, role uint32) {
}

func (server *ssntpNullServer) DisconnectNotify(uuid string, role uint32) {
}

func (server *ssntpNullServer) StatusNotify(uuid string, status Status, frame *Frame) {
	server.wg.Done()
}

func (server *ssntpNullServer) CommandNotify(uuid string, command Command, frame *Frame) {
	server.nCmds++
	if server.nCmds == server.b.N {
		server.nCmds = 0
		if server.done != nil {
			close(server.done)
		}
	}
}

func (server *ssntpNullServer) EventNotify(uuid string, event Event, frame *Frame) {
}

func (server *ssntpNullServer) ErrorNotify(uuid string, error Error, frame *Frame) {
}

type benchmarkClient struct {
	ssntp Client
	b     *testing.B
}

func (client *benchmarkClient) ConnectNotify() {
}

func (client *benchmarkClient) DisconnectNotify() {
}

func (client *benchmarkClient) StatusNotify(status Status, frame *Frame) {
}

func (client *benchmarkClient) CommandNotify(command Command, frame *Frame) {
}

func (client *benchmarkClient) EventNotify(event Event, frame *Frame) {
}

func (client *benchmarkClient) ErrorNotify(error Error, frame *Frame) {
}

func benchmarkSingleClient(b *testing.B, payloadSize int) {
	var serverConfig Config
	var clientConfig Config
	var server ssntpNullServer
	var client benchmarkClient
	payload := make([]byte, payloadSize)

	server.b = b
	server.nCmds = 0
	server.done = make(chan struct{})
	client.b = b

	serverConfig.Transport = *transport
	clientConfig.Transport = *transport

	time.Sleep(500 * time.Millisecond)
	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)
	client.ssntp.Dial(&clientConfig, &client)

	b.SetBytes((int64)(payloadSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.ssntp.SendCommand(START, payload)
	}

	<-server.done

	client.ssntp.Close()
	server.ssntp.Stop()
}

func benchmarkMultiClients(b *testing.B, payloadSize int, nClients int, nFrames int, delay int) {
	var serverConfig Config
	var server ssntpNullServer
	payload := make([]byte, payloadSize)

	server.b = b
	server.nCmds = 0

	for i := 0; i < payloadSize; i++ {
		payload[i] = (byte)(i)
	}

	serverConfig.Transport = *transport

	time.Sleep(500 * time.Millisecond)
	go server.ssntp.Serve(&serverConfig, &server)
	time.Sleep(500 * time.Millisecond)

	totalFrames := nClients * nFrames * b.N
	frameDelay := time.Duration(delay) * time.Millisecond
	b.SetBytes((int64)(totalFrames * payloadSize))
	b.ResetTimer()

	server.wg.Add(totalFrames)
	for n := 0; n < b.N; n++ {
		for i := 0; i < nClients; i++ {
			go func() {
				client := &benchmarkClient{
					b: b,
				}

				var clientConfig Config
				clientConfig.Transport = *transport

				client.ssntp.Dial(&clientConfig, client)
				for j := 0; j < nFrames; j++ {
					client.ssntp.SendStatus(READY, payload)
					time.Sleep(frameDelay)
				}
				client.ssntp.Close()
			}()
		}
	}

	server.wg.Wait()
	server.ssntp.Stop()

}

func Benchmark1Client0BFrames(b *testing.B) {
	benchmarkSingleClient(b, 0)
}

func Benchmark1Client512BFrames(b *testing.B) {
	benchmarkSingleClient(b, 512)
}

func Benchmark1Client65kBFrames(b *testing.B) {
	benchmarkSingleClient(b, 1<<16)
}

func Benchmark500Clients1Frame2kB(b *testing.B) {
	benchmarkMultiClients(b, 1<<11, 500, 1, 0)
}

func Benchmark100Clients100Frames2kBNoDelay(b *testing.B) {
	benchmarkMultiClients(b, 1<<11, 500, 1000, 0)
}

func Benchmark100Clients1Frame2kB(b *testing.B) {
	benchmarkMultiClients(b, 1<<11, 500, 1, 0)
}

func Benchmark100Clients100Frames65kB1msDelay(b *testing.B) {
	benchmarkMultiClients(b, 1<<16, 100, 1000, 1)
}

func BenchmarkDefaultMultiClientsMultiFrames(b *testing.B) {
	benchmarkMultiClients(b, *payloadSize, *clients, *frames, *delay)
}
