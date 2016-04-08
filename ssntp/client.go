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
	"crypto/tls"
	"fmt"
	"github.com/docker/distribution/uuid"
	"math/rand"
	"time"
)

// ClientNotifier is the SSNTP client notification interface.
// Any SSNTP client must implement this interface.
type ClientNotifier interface {
	// ConnectNotify notifies of a successful connection to an SSNTP server.
	// This notification is mostly useful for clients to know when they're
	// being re-connected to the SSNTP server.
	ConnectNotify()

	// DisconnectNotify notifies of a SSNTP server disconnection.
	// SSNTP Client implementations are not supposed to explicitely
	// reconnect, the SSNTP protocol will handle the reconnection.
	DisconnectNotify()

	// StatusNotify notifies of a pending status frame from the SSNTP server.
	StatusNotify(status Status, frame *Frame)

	// CommandNotify notifies of a pending command frame from the SSNTP server.
	CommandNotify(command Command, frame *Frame)

	// EventNotify notifies of a pending event frame from the SSNTP server.
	EventNotify(event Event, frame *Frame)

	// ErrorNotify notifies of a pending error frame from the SSNTP server.
	// Error frames are always related to the last sent frame.
	ErrorNotify(error Error, frame *Frame)
}

// Client is the SSNTP client structure.
// This is an SSNTP client handle to connect to and
// disconnect from an SSNTP server, and send SSNTP
// frames to it.
// It is an entirely opaque structure, only accessible through
// its public methods.
type Client struct {
	uuid       uuid.UUID
	lUUID      lockedUUID
	uris       []string
	role       uint32
	roleVerify bool
	tls        *tls.Config
	ntf        ClientNotifier
	transport  string
	port       uint32
	session    *session
	status     connectionStatus
	closed     chan struct{}

	log Logger

	trace *TraceConfig
}

func handleSSNTPServer(client *Client) {
	defer client.Close()

	for {
		client.ntf.ConnectNotify()

		for {
			client.log.Infof("Waiting for next frame\n")

			var frame Frame
			err := client.session.Read(&frame)
			if err != nil {
				client.status.Lock()
				if client.status.status == ssntpClosed {
					client.status.Unlock()
					return
				}
				client.status.Unlock()

				client.log.Errorf("Read error: %s\n", err)
				client.ntf.DisconnectNotify()
				break
			}

			switch (Type)(frame.Type) {
			case COMMAND:
				client.ntf.CommandNotify((Command)(frame.Operand), &frame)
			case STATUS:
				client.ntf.StatusNotify((Status)(frame.Operand), &frame)
			case EVENT:
				client.ntf.EventNotify((Event)(frame.Operand), &frame)
			case ERROR:
				client.ntf.ErrorNotify((Error)(frame.Operand), &frame)
			default:
				client.SendError(InvalidFrameType, nil)
			}
		}

		err := client.attemptDial()
		if err != nil {
			client.log.Errorf("%s", err)
			return
		}
	}
}

func (client *Client) sendConnect() (bool, error) {
	var connected ConnectFrame
	client.log.Infof("Sending CONNECT\n")

	connect := client.session.connectFrame()
	_, err := client.session.Write(connect)
	if err != nil {
		return true, err
	}

	client.log.Infof("Waiting for CONNECTED\n")
	err = client.session.Read(&connected)
	if err != nil {
		return true, err
	}

	client.log.Infof("Received CONNECTED frame:\n%s\n", connected)

	switch connected.Type {
	case STATUS:
		if connected.Operand != (uint8)(CONNECTED) {
			return true, fmt.Errorf("SSNTP Client: Invalid Connected frame")
		}
	case ERROR:
		if connected.Operand != (uint8)(ConnectionFailure) {
			return false, fmt.Errorf("SSNTP Client: Connection failure")
		}

		return true, fmt.Errorf("SSNTP Client: Connection error %s\n", (Error)(connected.Operand))

	default:
		return true, fmt.Errorf("SSNTP Client: Unknown frame type %d", connected.Type)
	}

	client.session.setDest(connected.Source[:16])
	if client.roleVerify == true {
		oidFound, err := verifyRole(client.session.conn, connected.Role)
		if oidFound == false {
			fmt.Printf("%s\n", err)
			client.SendError(ConnectionFailure, nil)
			return false, fmt.Errorf("SSNTP Client: Connection failure")
		}
	}

	client.status.Lock()
	client.status.status = ssntpConnected
	client.status.Unlock()

	client.log.Infof("Done with connection\n")

	return true, nil
}

func (client *Client) attemptDial() error {
	delays := []int64{5, 10, 20, 40}

	if len(client.uris) == 0 {
		return fmt.Errorf("No servers to connect to")
	}

	client.status.Lock()
	client.closed = make(chan struct{})
	client.status.Unlock()

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	for {
	URILoop:
		for _, uri := range client.uris {
			for d := 0; ; d++ {
				client.log.Infof("%s connecting to %s\n", client.uuid, uri)
				conn, err := tls.Dial(client.transport, uri, client.tls)

				client.status.Lock()
				if client.status.status == ssntpClosed {
					client.status.Unlock()
					return fmt.Errorf("Connection closed")
				}
				client.status.Unlock()

				if err != nil {
					client.log.Infof("Dial failed %s\n", err.Error())

					delay := r.Int63n(delays[d%len(delays)])
					delay++ // Avoid waiting for 0 seconds
					client.log.Errorf("Could not connect to %s (%s) - retrying in %d seconds\n", uri, err, delay)

					// Wait for delay before reconnecting or return if the client is closed
					select {
					case <-client.closed:
						return fmt.Errorf("Connection closed")
					case <-time.After(time.Duration(delay) * time.Second):
						break
					}

					continue
				}

				client.log.Infof("Connected\n")
				session := newSession(&client.uuid, client.role, 0, conn)
				client.session = session

				break URILoop

			}
		}

		if client.session == nil {
			continue
		}

		reconnect, err := client.sendConnect()
		if err != nil {
			// Dialed but could not connect, try again
			client.log.Errorf("%s", err)
			client.Close()
			if reconnect == true {
				continue
			} else {
				client.ntf.DisconnectNotify()
				return err
			}
		}

		// Dialed and connected, we can proceed
		break
	}

	return nil
}

// Dial attempts to connect to a SSNTP server, as specified by the config argument.
// Dial will try and retry to connect to this server and will wait for it to show
// up if it's temporarily unavailable. A client can be closed while it's still
// trying to connect to the SSNTP server, so that one can properly kill a client if
// e.g. no server will ever come alive.
// Once connected a separate routine will listen for server commands, statuses or
// errors and report them back through the SSNTP client notifier interface.
func (client *Client) Dial(config *Config, ntf ClientNotifier) error {
	if config == nil {
		return fmt.Errorf("SSNTP config missing")
	}

	client.status.Lock()

	if client.status.status == ssntpConnected || client.status.status == ssntpConnecting {
		client.status.Unlock()
		return fmt.Errorf("Client already connected")
	}

	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return fmt.Errorf("Client already closed")
	}

	client.status.status = ssntpConnecting

	client.status.Unlock()

	if len(config.UUID) == 0 {
		var err error
		client.lUUID, err = newUUID("client", config.Role)
		if err != nil {
			fmt.Printf("SSNTP ERROR: Client: Could not fetch a UUID, generating a random one (%s)\n", err)
			client.uuid = uuid.Generate()
		} else {
			client.uuid = client.lUUID.uuid
		}
	} else {
		uuid, _ := uuid.Parse(config.UUID)
		client.uuid = uuid
	}

	if config.Port != 0 {
		client.port = config.Port
	} else {
		client.port = port
	}

	if len(config.URI) == 0 {
		client.uris = append(client.uris, fmt.Sprintf("%s:%d", defaultURL, client.port))
	} else {
		client.uris = append(client.uris, fmt.Sprintf("%s:%d", config.URI, client.port))
	}

	if len(config.Transport) == 0 {
		client.transport = "tcp"
	} else {
		if config.Transport != "tcp" && config.Transport != "unix" {
			client.transport = "tcp"
		} else {
			client.transport = config.Transport
		}
	}

	client.role = config.Role
	client.roleVerify = config.RoleVerification

	if len(config.CAcert) == 0 {
		config.CAcert = defaultCA
	}

	if len(config.Cert) == 0 {
		config.Cert = defaultClientCert
	}

	if config.Log == nil {
		client.log = errLog
	} else {
		client.log = config.Log
	}

	client.trace = config.Trace
	client.ntf = ntf
	client.tls = prepareTLSConfig(config, false)

	err := client.attemptDial()
	if err != nil {
		client.log.Errorf("%s", err)
		return err
	}

	go handleSSNTPServer(client)

	return nil
}

// Close terminates the client connection.
func (client *Client) Close() {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return
	}

	if client.session != nil {
		client.session.conn.Close()
	}
	client.status.status = ssntpClosed
	if client.closed != nil {
		close(client.closed)
	}
	client.status.Unlock()

	freeUUID(client.lUUID)
}

func (client *Client) sendCommand(cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.commandFrame(cmd, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendStatus(status Status, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.statusFrame(status, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendEvent(event Event, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.eventFrame(event, payload, trace)

	return session.Write(frame)
}

func (client *Client) sendError(error Error, payload []byte, trace *TraceConfig) (int, error) {
	client.status.Lock()
	if client.status.status == ssntpClosed {
		client.status.Unlock()
		return -1, fmt.Errorf("Client not connected")
	}
	client.status.Unlock()

	session := client.session
	frame := session.errorFrame(error, payload, trace)

	return session.Write(frame)
}

// SendCommand sends a specific command and its payload to the SSNTP server.
func (client *Client) SendCommand(cmd Command, payload []byte) (int, error) {
	return client.sendCommand(cmd, payload, client.trace)
}

// SendStatus sends a specific status and its payload to the SSNTP server.
func (client *Client) SendStatus(status Status, payload []byte) (int, error) {
	return client.sendStatus(status, payload, client.trace)
}

// SendEvent sends a specific status and its payload to the SSNTP server.
func (client *Client) SendEvent(event Event, payload []byte) (int, error) {
	return client.sendEvent(event, payload, client.trace)
}

// SendError sends an error back to the SSNTP server.
// This is just for notification purposes, to let e.g. the server know that
// it sent an unexpected frame.
func (client *Client) SendError(error Error, payload []byte) (int, error) {
	return client.sendError(error, payload, client.trace)
}

// SendTracedCommand sends a specific command and its payload to the SSNTP server.
// The SSNTP command frame will be traced according to the trace argument.
func (client *Client) SendTracedCommand(cmd Command, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendCommand(cmd, payload, trace)
}

// SendTracedStatus sends a specific status and its payload to the SSNTP server.
// The SSNTP status frame will be traced according to the trace argument.
func (client *Client) SendTracedStatus(status Status, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendStatus(status, payload, trace)
}

// SendTracedEvent sends a specific status and its payload to the SSNTP server.
// The SSNTP event frame will be traced according to the trace argument.
func (client *Client) SendTracedEvent(event Event, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendEvent(event, payload, trace)
}

// SendTracedError sends an error back to the SSNTP server.
// This is just for notification purposes, to let e.g. the server know that
// it sent an unexpected frame.
// The SSNTP error frame will be traced according to the trace argument.
func (client *Client) SendTracedError(error Error, payload []byte, trace *TraceConfig) (int, error) {
	return client.sendError(error, payload, trace)
}

// UUID exports the SSNTP client Universally Unique ID.
func (client *Client) UUID() string {
	return client.uuid.String()
}
