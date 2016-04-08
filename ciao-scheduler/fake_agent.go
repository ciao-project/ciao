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

func (client *ssntpClient) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

func (client *ssntpClient) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	payload := frame.Payload
	client.nCommands++

	if command == ssntp.START {
		var cmd payloads.Start
		var memReqMB int
		err := yaml.Unmarshal(payload, &cmd)
		if err != nil {
			fmt.Println("bad START workload yaml from Controller")
			return
		}
		statsMutex.Lock()
		for idx := range cmd.Start.RequestedResources {
			if cmd.Start.RequestedResources[idx].Type == payloads.MemMB {
				memReqMB = cmd.Start.RequestedResources[idx].Value
			}
		}
		stats.MemAvailableMB -= memReqMB
		statsMutex.Unlock()
	}
}

func (client *ssntpClient) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
}

func (client *ssntpClient) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
	fmt.Printf("ERROR for %s\n", client.name)
}

var stats payloads.Stat
var statsMutex sync.Mutex

func fakeStatisticsThread(config *ssntp.Config, nFrames int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &ssntpClient{
		name:      "Ciao fake Agent",
		nCommands: 0,
	}

	fmt.Printf("----- Client [%s] delay [random] frames [%d] -----\n", client.ssntp.UUID()[:8], nFrames)

	if client.ssntp.Dial(config, client) != nil {
		fmt.Printf("Could not connect to an SSNTP server\n")
		return
	}
	fmt.Printf("Client [%s]: Connected\n", client.ssntp.UUID()[:8])

	//pretend it takes some time to start up a node
	time.Sleep(time.Duration(1) * time.Second)
	fmt.Printf("...warming up\n")
	time.Sleep(time.Duration(3) * time.Second)

	//dummy initial stats
	statsMutex.Lock()
	stats.Init()
	stats.NodeUUID = client.ssntp.UUID()
	stats.MemTotalMB = 3896     // 4096 - 200 overhead
	stats.MemAvailableMB = 3896 // start with "all"
	stats.Load = 0
	payload, err := yaml.Marshal(&stats)
	statsMutex.Unlock()
	if err != nil {
		fmt.Printf("Could not create STATS yaml: %s\n", err)
		return
	}
	time.Sleep(time.Duration(1) * time.Second)

	//and now we're READY
	sentFrames := 0
	_, err = client.ssntp.SendStatus(ssntp.READY, payload)
	if err != nil {
		fmt.Printf("Could not send READY: %s\n", err)
		return
	} else {
		sentFrames++
	}

	for i := 0; i < nFrames; i++ {
		// 1..~2 seconds delay between commands
		delay := rand.Intn(2000)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		fmt.Printf("Client [%s]: delay %d\n", client.ssntp.UUID()[:8], delay)

		statsMutex.Lock()
		payload, err = yaml.Marshal(&stats)
		statsMutex.Unlock()
		if err != nil {
			fmt.Printf("Could not create READY yaml: %s\n", err)
		}

		fmt.Printf("payload[%d]:%s\n", len(payload), payload)

		_, err = client.ssntp.SendStatus(ssntp.READY, payload)
		if err != nil {
			fmt.Printf("Could not send READY: %s\n", err)
		} else {
			sentFrames++
		}

		// "adjust" mem & load stats
		memdelta := rand.Intn(100)
		loaddelta := rand.Intn(10)
		statsMutex.Lock()
		if rand.Intn(2) == 0 {
			stats.MemAvailableMB -= memdelta
		} else {
			stats.MemAvailableMB += memdelta
		}
		if stats.MemAvailableMB > stats.MemTotalMB {
			stats.MemAvailableMB = stats.MemTotalMB
		}
		if rand.Intn(2) == 0 {
			stats.Load -= loaddelta
		} else {
			stats.Load += loaddelta
		}
		if stats.Load < 0 {
			stats.Load = 0
		}
		statsMutex.Unlock()
	}

	fmt.Printf("Client [%s]: Done\n", client.ssntp.UUID()[:8])

	time.Sleep(time.Duration(2000) * time.Millisecond)

	client.ssntp.Close()

	fmt.Printf("Sent %d commands, received %d\n", sentFrames, client.nCommands)
}

func main() {
	var serverURL = flag.String("ur", "localhost", "Server URL")
	var cert = flag.String("cert", "/etc/pki/ciao/cert-client-localhost.pem", "Client certificate")
	var CAcert = flag.String("cacert", "/etc/pki/ciao/CAcert-server-localhost.pem", "CA certificate")
	var nFrames = flag.Int("frames", 100, "Number of frames to send")
	var cpuprofile = flag.String("cpuprofile", "", "Write cpu profile to file")
	var role ssntp.Role = ssntp.AGENT
	var config ssntp.Config
	var wg sync.WaitGroup

	flag.Var(&role, "role", "Agent client role")
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
	go fakeStatisticsThread(&config, *nFrames, &wg)

	wg.Wait()
}
