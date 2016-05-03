/*
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
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/01org/ciao/payloads"
	"gopkg.in/yaml.v2"

	"github.com/golang/glog"
)

const (
	qemuEfiFw  = "/usr/share/qemu/OVMF.fd"
	seedImage  = "seed.iso"
	ciaoImage  = "ciao.iso"
	imagesPath = "/var/lib/ciao/images"
	vcTries    = 10
)

var virtualSizeRegexp *regexp.Regexp
var pssRegexp *regexp.Regexp

func init() {
	virtualSizeRegexp = regexp.MustCompile(`virtual size:.*\(([0-9]+) bytes\)`)
	pssRegexp = regexp.MustCompile(`^Pss:\s*([0-9]+)`)
}

type qemu struct {
	cfg            *vmConfig
	instanceDir    string
	vcPort         int
	pid            int
	prevCPUTime    int64
	prevSampleTime time.Time
	isoPath        string
	ciaoISOPath    string
}

func (q *qemu) init(cfg *vmConfig, instanceDir string) {
	q.cfg = cfg
	q.instanceDir = instanceDir
	q.isoPath = path.Join(instanceDir, seedImage)
	q.ciaoISOPath = path.Join(instanceDir, ciaoImage)
}

func (q *qemu) imageInfo(imagePath string) (imageSizeMB int, err error) {
	imageSizeMB = -1

	params := make([]string, 0, 8)
	params = append(params, "info")
	params = append(params, imagePath)

	cmd := exec.Command("qemu-img", params...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		glog.Errorf("Unable to read output from qemu-img: %v", err)
		return -1, err
	}

	err = cmd.Start()
	if err != nil {
		_ = stdout.Close()
		glog.Errorf("Unable start qemu-img: %v", err)
		return -1, err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() && imageSizeMB == -1 {
		line := scanner.Text()
		matches := virtualSizeRegexp.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		if len(matches) < 2 {
			glog.Warningf("Unable to find image size from: %s",
				line)
			break
		}

		sizeInBytes, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			glog.Warningf("Unable to parse image size from: %s",
				matches[1])
			break
		}

		size := sizeInBytes / (1000 * 1000)
		if size > int64((^uint(0))>>1) {
			glog.Warningf("Unexpectedly large disk size found: %d MB",
				size)
			break
		}

		imageSizeMB = int(size)
		if int64(imageSizeMB)*1000*1000 < sizeInBytes {
			imageSizeMB++
		}
	}

	err = cmd.Wait()
	if err != nil {
		glog.Warningf("qemu-img returned an error: %v", err)
		if imageSizeMB != -1 {
			glog.Warning("But we already parsed the image size, so we don't care")
			err = nil
		}
	}

	return imageSizeMB, err
}

func createCloudInitISO(instanceDir, isoPath string, cfg *vmConfig, userData, metaData []byte) error {

	configDrivePath := path.Join(instanceDir, "clr-cloud-init")
	dataDirPath := path.Join(configDrivePath, "openstack", "latest")
	metaDataPath := path.Join(dataDirPath, "meta_data.json")
	userDataPath := path.Join(dataDirPath, "user_data")

	defer func() {
		_ = os.RemoveAll(configDrivePath)
	}()

	err := os.MkdirAll(dataDirPath, 0755)
	if err != nil {
		glog.Errorf("Unable to create config drive directory %s", dataDirPath)
		return err
	}

	if len(metaData) == 0 {
		defaultMeta := fmt.Sprintf("{\n  \"uuid\": %q,\n  \"hostname\": %[1]q\n}\n", cfg.Instance)
		metaData = []byte(defaultMeta)
	}

	err = ioutil.WriteFile(metaDataPath, metaData, 0644)
	if err != nil {
		glog.Errorf("Unable to create %s", metaDataPath)
		return err
	}

	err = ioutil.WriteFile(userDataPath, userData, 0644)
	if err != nil {
		glog.Errorf("Unable to create %s", userDataPath)
		return err
	}

	cmd := exec.Command("xorriso", "-as", "mkisofs", "-R", "-V", "config-2", "-o", isoPath,
		configDrivePath)
	err = cmd.Run()
	if err != nil {
		glog.Errorf("Unable to create cloudinit iso image %v", err)
		return err
	}

	glog.Infof("ISO image %s created", isoPath)

	return nil
}

func createCiaoISO(instanceDir, isoPath string) error {
	ciaoDrivePath := path.Join(instanceDir, "ciao")
	ciaoPath := path.Join(ciaoDrivePath, "ciao.yaml")

	defer func() {
		_ = os.RemoveAll(ciaoDrivePath)
	}()

	err := os.MkdirAll(ciaoDrivePath, 0755)
	if err != nil {
		glog.Errorf("Unable to create ciao drive directory %s", ciaoDrivePath)
		return err
	}

	config := payloads.CNCIInstanceConfig{SchedulerAddr: serverURL}
	y, err := yaml.Marshal(&config)
	if err != nil {
		glog.Errorf("Unable to create yaml ciao file %s", err)
		return err
	}

	err = ioutil.WriteFile(ciaoPath, y, 0644)
	if err != nil {
		glog.Errorf("Unable to create %s", ciaoPath)
		return err
	}

	cmd := exec.Command("xorriso", "-as", "mkisofs", "-R", "-V", "ciao", "-o", isoPath,
		ciaoPath)
	err = cmd.Run()
	if err != nil {
		glog.Errorf("Unable to create ciao iso image %v", err)
		return err
	}

	glog.Infof("Ciao ISO image %s created", isoPath)

	return nil
}

func (q *qemu) createRootfs() error {
	vmImage := path.Join(q.instanceDir, "image.qcow2")
	backingImage := path.Join(imagesPath, q.cfg.Image)
	glog.Infof("Creating qcow image from %s backing %s", vmImage, backingImage)

	params := make([]string, 0, 32)
	params = append(params, "create", "-f", "qcow2", "-o", "backing_file="+backingImage,
		vmImage)
	if q.cfg.Disk > 0 {
		diskSize := fmt.Sprintf("%dM", q.cfg.Disk)
		params = append(params, diskSize)
	}

	cmd := exec.Command("qemu-img", params...)
	return cmd.Run()
}

func (q *qemu) checkBackingImage() error {
	backingImage := path.Join(imagesPath, q.cfg.Image)
	_, err := os.Stat(backingImage)
	if err != nil {
		return fmt.Errorf("Backing Image does not exist: %v", err)
	}

	if q.cfg.Disk != 0 {
		minSizeMB, err := getMinImageSize(q, backingImage)
		if err != nil {
			return fmt.Errorf("Unable to determine image size: %v", err)
		}

		if minSizeMB != -1 && minSizeMB > q.cfg.Disk {
			glog.Warningf("Requested disk size (%dM) is smaller than minimum image size (%dM).  Defaulting to min size", q.cfg.Disk, minSizeMB)
			q.cfg.Disk = minSizeMB
		}
	}

	return nil
}

func (q *qemu) downloadBackingImage() error {
	return fmt.Errorf("Not supported yet!")
}

func (q *qemu) createImage(bridge string, userData, metaData []byte) error {
	err := createCloudInitISO(q.instanceDir, q.isoPath, q.cfg, userData, metaData)
	if err != nil {
		glog.Errorf("Unable to create iso image %v", err)
		return err
	}

	if q.cfg.NetworkNode {
		err = createCiaoISO(q.instanceDir, q.ciaoISOPath)
		if err != nil {
			return err
		}
	}

	return q.createRootfs()
}

func (q *qemu) deleteImage() error {
	return nil
}

func cleanupFds(fds []*os.File, numFds int) {

	maxFds := len(fds)

	if numFds < maxFds {
		maxFds = numFds
	}

	for i := 0; i < maxFds; i++ {
		_ = fds[i].Close()
	}
}

func computeMacvtapParam(vnicName string, mac string, queues int) ([]string, []*os.File, error) {

	fds := make([]*os.File, queues)
	params := make([]string, 0, 8)

	ifIndexPath := path.Join("/sys/class/net", vnicName, "ifindex")
	fip, err := os.Open(ifIndexPath)
	if err != nil {
		glog.Errorf("Failed to determine tap ifname: %s", err)
		return nil, nil, err
	}
	defer func() { _ = fip.Close() }()

	scan := bufio.NewScanner(fip)
	if !scan.Scan() {
		glog.Error("Unable to read tap index")
		return nil, nil, fmt.Errorf("Unable to read tap index")
	}

	i, err := strconv.Atoi(scan.Text())
	if err != nil {
		glog.Errorf("Failed to determine tap ifname: %s", err)
		return nil, nil, err
	}

	//mq support
	var fdParam bytes.Buffer
	fdSeperator := ""
	for q := 0; q < queues; q++ {

		tapDev := fmt.Sprintf("/dev/tap%d", i)

		f, err := os.OpenFile(tapDev, os.O_RDWR, 0666)
		if err != nil {
			glog.Errorf("Failed to open tap device %s: %s", tapDev, err)
			cleanupFds(fds, q)
			return nil, nil, err
		}
		fds[q] = f
		/*
		   3, what do you mean 3.  Well, it turns out that files passed to child
		   processes via cmd.ExtraFiles have different fds in the child and the
		   parent.  In the child the fds are determined by the file's position
		   in the ExtraFiles array + 3.
		*/

		// bytes.WriteString does not return an error
		_, _ = fdParam.WriteString(fmt.Sprintf("%s%d", fdSeperator, q+3))
		fdSeperator = ":"
	}

	netdev := fmt.Sprintf("type=tap,fds=%s,id=%s,vhost=on", fdParam.String(), vnicName)
	device := fmt.Sprintf("virtio-net-pci,netdev=%s,mq=on,vectors=%d,mac=%s", vnicName, 32, mac)
	params = append(params, "-netdev", netdev)
	params = append(params, "-device", device)
	return params, fds, nil
}

func computeTapParam(vnicName string, mac string) ([]string, error) {
	params := make([]string, 0, 8)
	net1Param := fmt.Sprintf("nic,model=virtio,macaddr=%s", mac)
	net2Param := fmt.Sprintf("tap,ifname=%s,script=no,downscript=no", vnicName)
	params = append(params, "-net", net1Param)
	params = append(params, "-net", net2Param)
	return params, nil
}

func launchQemu(params []string, fds []*os.File) (string, error) {
	errStr := ""
	cmd := exec.Command("qemu-system-x86_64", params...)
	if fds != nil {
		glog.Infof("Adding extra file %v", fds)
		cmd.ExtraFiles = fds
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	glog.Infof("launching qemu with: %v", params)

	err := cmd.Run()
	if err != nil {
		glog.Errorf("Unable to launch qemu: %v", err)
		errStr = stderr.String()
		glog.Error(errStr)
	}
	return errStr, err
}

func launchQemuWithNC(params []string, fds []*os.File, ipAddress string) (int, error) {
	var err error

	tries := 0
	params = append(params, "-display", "none", "-vga", "none")
	params = append(params, "-device", "isa-serial,chardev=gnc0", "-chardev", "")
	port := 0
	for ; tries < vcTries; tries++ {
		port = uiPortGrabber.grabPort()
		if port == 0 {
			break
		}
		ncString := "socket,port=%d,host=%s,server,id=gnc0,server,nowait"
		params[len(params)-1] = fmt.Sprintf(ncString, port, ipAddress)
		var errStr string
		errStr, err = launchQemu(params, fds)
		if err == nil {
			glog.Info("============================================")
			glog.Infof("Connect to vm with netcat %s %d", ipAddress, port)
			glog.Info("============================================")
			break
		}

		lowErr := strings.ToLower(errStr)
		if !strings.Contains(lowErr, "socket") {
			uiPortGrabber.releasePort(port)
			break
		}
	}

	if port == 0 || (err != nil && tries == vcTries) {
		glog.Warning("Failed to launch qemu due to chardev error.  Relaunching without virtual console")
		_, err = launchQemu(params[:len(params)-4], fds)
	}

	return port, err
}

func launchQemuWithSpice(params []string, fds []*os.File, ipAddress string) (int, error) {
	var err error

	tries := 0
	params = append(params, "-spice", "")
	port := 0
	for ; tries < vcTries; tries++ {
		port = uiPortGrabber.grabPort()
		if port == 0 {
			break
		}
		params[len(params)-1] = fmt.Sprintf("port=%d,addr=%s,disable-ticketing", port, ipAddress)
		var errStr string
		errStr, err = launchQemu(params, fds)
		if err == nil {
			glog.Info("============================================")
			glog.Infof("Connect to vm with spicec -h %s -p %d", ipAddress, port)
			glog.Info("============================================")
			break
		}

		// Not great I know, but it's the only way to figure out if spice is at fault
		lowErr := strings.ToLower(errStr)
		if !strings.Contains(lowErr, "spice") {
			uiPortGrabber.releasePort(port)
			break
		}
	}

	if port == 0 || (err != nil && tries == vcTries) {
		glog.Warning("Failed to launch qemu due to spice error.  Relaunching without virtual console")
		params = append(params[:len(params)-2], "-display", "none", "-vga", "none")
		_, err = launchQemu(params, fds)
	}

	return port, err
}

func (q *qemu) startVM(vnicName, ipAddress string) error {

	var fds []*os.File

	glog.Info("Launching qemu")

	vmImage := path.Join(q.instanceDir, "image.qcow2")
	qmpSocket := path.Join(q.instanceDir, "socket")
	fileParam := fmt.Sprintf("file=%s,if=virtio,aio=threads,format=qcow2", vmImage)
	//BUG(markus): Should specify media type here
	isoParam := fmt.Sprintf("file=%s,if=virtio", q.isoPath)
	qmpParam := fmt.Sprintf("unix:%s,server,nowait", qmpSocket)

	params := make([]string, 0, 32)
	params = append(params, "-drive", fileParam)
	params = append(params, "-drive", isoParam)
	if q.cfg.NetworkNode {
		ciaoParam := fmt.Sprintf("file=%s,if=virtio", q.ciaoISOPath)
		params = append(params, "-drive", ciaoParam)
	}

	if vnicName != "" {
		if q.cfg.NetworkNode {
			var err error
			var macvtapParam []string
			//TODO: @mcastelino get from scheduler/controller
			numQueues := 4
			macvtapParam, fds, err = computeMacvtapParam(vnicName, q.cfg.VnicMAC, numQueues)
			if err != nil {
				return err
			}
			defer cleanupFds(fds, len(fds))
			params = append(params, macvtapParam...)
		} else {
			tapParam, err := computeTapParam(vnicName, q.cfg.VnicMAC)
			if err != nil {
				return err
			}
			params = append(params, tapParam...)
		}
	} else {
		params = append(params, "-net", "nic,model=virtio")
		params = append(params, "-net", "user")
	}

	params = append(params, "-enable-kvm")
	params = append(params, "-cpu", "host")
	params = append(params, "-daemonize")
	params = append(params, "-qmp", qmpParam)

	if q.cfg.Mem > 0 {
		memoryParam := fmt.Sprintf("%d", q.cfg.Mem)
		params = append(params, "-m", memoryParam)
	}
	if q.cfg.Cpus > 0 {
		cpusParam := fmt.Sprintf("cpus=%d", q.cfg.Cpus)
		params = append(params, "-smp", cpusParam)
	}

	if !q.cfg.Legacy {
		params = append(params, "-bios", qemuEfiFw)
	}

	var err error

	if !launchWithUI.Enabled() {
		params = append(params, "-display", "none", "-vga", "none")
		_, err = launchQemu(params, fds)
	} else if launchWithUI.String() == "spice" {
		var port int
		port, err = launchQemuWithSpice(params, fds, ipAddress)
		if err == nil {
			q.vcPort = port
		}
	} else {
		var port int
		port, err = launchQemuWithNC(params, fds, ipAddress)
		if err == nil {
			q.vcPort = port
		}
	}

	if err != nil {
		return err
	}

	glog.Info("Launched VM")

	return nil
}

func (q *qemu) lostVM() {
	if launchWithUI.Enabled() {
		glog.Infof("Releasing VC Port %d", q.vcPort)
		uiPortGrabber.releasePort(q.vcPort)
		q.vcPort = 0
	}
	q.pid = 0
	q.prevCPUTime = -1
}

func qmpConnect(qmpChannel chan string, instance, instanceDir string, closedCh chan struct{},
	connectedCh chan struct{}, wg *sync.WaitGroup, boot bool) {
	var conn net.Conn

	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
		if closedCh != nil {
			close(closedCh)
		}
		glog.Infof("Monitor function for %s exitting", instance)
		wg.Done()
	}()

	qmpSocket := path.Join(instanceDir, "socket")
	conn, err := net.DialTimeout("unix", qmpSocket, time.Second*30)
	if err != nil {
		glog.Errorf("Unable to open qmp socket for instance %s: %v", instance, err)
		return
	}

	scanner := bufio.NewScanner(conn)
	_, err = fmt.Fprintln(conn, "{ \"execute\": \"qmp_capabilities\" }")
	if err != nil {
		glog.Errorf("Unable to send qmp_capabilities to instance %s: %v", instance, err)
		return
	}

	/* TODO check return value and implement timeout */

	if !scanner.Scan() {
		glog.Errorf("qmp_capabilities failed on instance %s", instance)
		return
	}

	close(connectedCh)

	eventCh := make(chan string)
	go func() {
		for scanner.Scan() {
			text := scanner.Text()
			if glog.V(1) {
				glog.Info(text)
			}
			eventCh <- scanner.Text()
		}
		glog.Infof("Quitting %s read Loop", instance)
		close(eventCh)
	}()

	waitForShutdown := false
	quitting := false

DONE:
	for {
		select {
		case cmd, ok := <-qmpChannel:
			if !ok {
				qmpChannel = nil
				if !waitForShutdown {
					break DONE
				} else {
					quitting = true
				}
			}
			if cmd == virtualizerStopCmd {
				glog.Info("Sending STOP")
				_, err = fmt.Fprintln(conn, "{ \"execute\": \"quit\" }")
				if err != nil {
					glog.Errorf("Unable to send power down command to %s: %v\n", instance, err)
				} else {
					waitForShutdown = true
				}
			}
		case event, ok := <-eventCh:
			if !ok {
				close(closedCh)
				closedCh = nil
				eventCh = nil
				waitForShutdown = false
				if quitting {
					glog.Info("Lost connection to qemu domain socket")
					break DONE
				} else {
					glog.Warning("Lost connection to qemu domain socket")
				}
				continue
			}
			if waitForShutdown == true && strings.Contains(event, "return") {
				waitForShutdown = false
				if quitting {
					break DONE
				}
			}
		}
	}

	_ = conn.Close()
	conn = nil

	/* Readloop could be blocking on a send */

	if eventCh != nil {
		for range eventCh {
		}
	}

	glog.Infof("Quitting Monitor Loop for %s\n", instance)
}

/* closedCh is closed by the monitor go routine when it loses connection to the domain socket, basically,
   indicating that the VM instance has shut down.  The instance go routine is expected to close the
   qmpChannel to force the monitor go routine to exit.

   connectedCh is closed when we successfully connect to the domain socket, inidcating that the
   VM instance is running.
*/

func (q *qemu) monitorVM(closedCh chan struct{}, connectedCh chan struct{},
	wg *sync.WaitGroup, boot bool) chan string {
	qmpChannel := make(chan string)
	wg.Add(1)
	go qmpConnect(qmpChannel, q.cfg.Instance, q.instanceDir, closedCh, connectedCh, wg, boot)
	return qmpChannel
}

func computeInstanceDiskspace(instanceDir string) int {
	vmImage := path.Join(instanceDir, "image.qcow2")
	fi, err := os.Stat(vmImage)
	if err != nil {
		return -1
	}
	return int(fi.Size() / 1000000)
}

func (q *qemu) stats() (disk, memory, cpu int) {
	disk = computeInstanceDiskspace(q.instanceDir)
	memory = -1
	cpu = -1

	if q.pid == 0 {
		return
	}

	memory = computeProcessMemUsage(q.pid)
	if q.cfg == nil {
		return
	}

	cpuTime := computeProcessCPUTime(q.pid)
	now := time.Now()
	if q.prevCPUTime != -1 {
		cpu = int((100 * (cpuTime - q.prevCPUTime) /
			now.Sub(q.prevSampleTime).Nanoseconds()))
		if q.cfg.Cpus > 1 {
			cpu /= q.cfg.Cpus
		}
		// if glog.V(1) {
		//     glog.Infof("cpu %d%%\n", cpu)
		// }
	}
	q.prevCPUTime = cpuTime
	q.prevSampleTime = now

	return
}

func (q *qemu) connected() {
	qmpSocket := path.Join(q.instanceDir, "socket")
	var buf bytes.Buffer
	cmd := exec.Command("fuser", qmpSocket)
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		glog.Errorf("Failed to run fuser: %v", err)
		return
	}

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		pidString := strings.TrimSpace(scanner.Text())
		pid, err := strconv.Atoi(pidString)
		if err != nil {
			continue
		}

		if pid != 0 && pid != os.Getpid() {
			glog.Infof("PID of qemu for instance %s is %d", q.instanceDir, pid)
			q.pid = pid
			break
		}
	}

	if q.pid == 0 {
		glog.Errorf("Unable to determine pid for %s", q.instanceDir)
	}
	q.prevCPUTime = -1
}

func qemuKillInstance(instanceDir string) {
	var conn net.Conn

	qmpSocket := path.Join(instanceDir, "socket")
	conn, err := net.DialTimeout("unix", qmpSocket, time.Second*30)
	if err != nil {
		return
	}

	defer func() { _ = conn.Close() }()

	_, err = fmt.Fprintln(conn, "{ \"execute\": \"qmp_capabilities\" }")
	if err != nil {
		glog.Errorf("Unable to send qmp_capabilities to instance %s: %v", instanceDir, err)
		return
	}

	glog.Infof("Powering Down %s", instanceDir)

	_, err = fmt.Fprintln(conn, "{ \"execute\": \"quit\" }")
	if err != nil {
		glog.Errorf("Unable to send power down command to %s: %v\n", instanceDir, err)
	}

	// Keep reading until the socket fails.  If we close the socket straight away, qemu does not
	// honour our quit command.

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
	}

	return
}
