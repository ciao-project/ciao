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
// +build ignore

package main

import (
	"flag"
	"fmt"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
	"math/rand"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

type ssntpClient struct {
	ssntp     ssntp.Client
	name      string
	nCommands int
}

func (client *ssntpClient) ConnectNotify() {
	fmt.Printf("%s connected\n", client.name)
}

func (client *ssntpClient) DisconnectNotify() {
	fmt.Printf("%s disconnected\n", client.name)
}

func (client *ssntpClient) StatusNotify(status ssntp.Status, payload *ssntp.Frame) {
	fmt.Printf("STATUS %s for %s\n", status, client.name)
}

func (client *ssntpClient) CommandNotify(command ssntp.Command, payload *ssntp.Frame) {
	client.nCommands++
}

func (client *ssntpClient) EventNotify(event ssntp.Event, payload *ssntp.Frame) {
}

func (client *ssntpClient) ErrorNotify(error ssntp.Error, payload *ssntp.Frame) {
	fmt.Printf("ERROR for %s\n", client.name)
}

func fakeControllerCommandSenderThread(config *ssntp.Config, n int, nFrames int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &ssntpClient{
		name:      "Ciao fake Controller command sender",
		nCommands: 0,
	}

	// set up a dummy START command
	reqVcpus := payloads.RequestedResource{
		Type:      "vcpus",
		Value:     2,
		Mandatory: true,
	}
	reqMem := payloads.RequestedResource{
		Type:      "mem_mb",
		Value:     256,
		Mandatory: true,
	}
	reqDisk := payloads.RequestedResource{
		Type:      "disk_mb",
		Value:     10000,
		Mandatory: true,
	}
	estVcpus := payloads.EstimatedResource{
		Type:  "vcpus",
		Value: 1,
	}
	estMem := payloads.EstimatedResource{
		Type:  "mem_mb",
		Value: 128,
	}
	estDisk := payloads.EstimatedResource{
		Type:  "disk_mb",
		Value: 4096,
	}
	var cmd payloads.Start
	cmd.Start.InstanceUUID = "c73322e8-d5fe-4d57-874c-dcee4fd368cd"
	cmd.Start.ImageUUID = "b265f62b-e957-47fd-a0a2-6dc261c7315c"
	cmd.Start.RequestedResources = append(cmd.Start.RequestedResources, reqVcpus)
	cmd.Start.RequestedResources = append(cmd.Start.RequestedResources, reqMem)
	cmd.Start.RequestedResources = append(cmd.Start.RequestedResources, reqDisk)
	cmd.Start.EstimatedResources = append(cmd.Start.EstimatedResources, estVcpus)
	cmd.Start.EstimatedResources = append(cmd.Start.EstimatedResources, estMem)
	cmd.Start.EstimatedResources = append(cmd.Start.EstimatedResources, estDisk)
	cmd.Start.FWType = payloads.EFI
	cmd.Start.InstancePersistence = payloads.Host

	payload, err := yaml.Marshal(&cmd)
	if err != nil {
		fmt.Printf("Could not create START workload yaml: %s\n", err)
	}

	if client.ssntp.Dial(config, client) != nil {
		fmt.Printf("Could not connect to an SSNTP server\n")
		return
	}
	fmt.Printf("Client [%d]: Connected\n", n)

	sentFrames := 0
	for i := 0; i < nFrames; i++ {
		// 1..10 seconds delay between commands
		delay := rand.Intn(10)
		fmt.Printf("Client [%d]: delay %d\n", n, delay)
		time.Sleep(time.Duration(delay) * time.Second)

		_, err := client.ssntp.SendCommand(ssntp.START, payload)
		if err != nil {
			fmt.Printf("Could not send START command: %s\n", err)
		} else {
			fmt.Printf("Client [%d]: sent START command\n", n)
		}
		if err == nil {
			sentFrames++
		}
	}

	fmt.Printf("Client [%d]: Done\n", n)

	client.ssntp.Close()

	fmt.Printf("Sent %d commands, received %d\n", sentFrames, client.nCommands)
}
func fakeControllerStatusReceiverThread(config *ssntp.Config, n int, nFrames int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &ssntpClient{
		name:      "Ciao fake Controller status receiver",
		nCommands: 0,
	}

	if client.ssntp.Dial(config, client) != nil {
		fmt.Printf("Could not connect to an SSNTP server\n")
		return
	}
	fmt.Printf("Controller Status Receiver Therad Client [%d]: Connected\n", n)

	//TODO: how to receive a forwarded STATS command
	for {
		//TODO: validate a STATS command was received

		//...for now do nothing
		time.Sleep(time.Duration(1) * time.Second)
	}

	fmt.Printf("Controller Status Receiver Therad Client [%d]: Done\n", n)

	client.ssntp.Close()

	//fmt.Printf("Sent %d commands, received %d\n", sentFrames, client.nCommands)
}

func main() {
	var serverURL = flag.String("url", "localhost", "Server URL")
	var cert = flag.String("cert", "/etc/pki/ciao/cert-client-localhost.pem", "Client certificate")
	var CAcert = flag.String("cacert", "/etc/pki/ciao/CAcert-server-localhost.pem", "CA certificate")
	var nFrames = flag.Int("frames", 10, "Number of frames to send")
	var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	var role ssntp.Role = ssntp.Controller
	var config ssntp.Config
	var wg sync.WaitGroup

	flag.Var(&role, "role", "Controller client role")
	flag.Parse()

	config.URI = *serverURL
	config.CAcert = *CAcert
	config.Cert = *cert
	config.Role = uint32(role)
	//	config.Trace = os.Stdout
	//	config.Error = os.Stdout

	if len(*cpuprofile) != 0 {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Print(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	wg.Add(1)
	go fakeControllerCommandSenderThread(&config, 1, *nFrames, &wg)

	//	wg.Add(1)
	//	go fakeControllerStatusReceiverThread(&config, i, *nFrames, &wg)

	wg.Wait()
}
