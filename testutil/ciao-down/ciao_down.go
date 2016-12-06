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

/* TODO

5. Install kernel
12. Make most output from osprepare optional
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s [prepare|start|stop|quit|status|connect|delete]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "- prepare : creates a new VM")
		fmt.Fprintln(os.Stderr, "- start : boots a stopped VM")
		fmt.Fprintln(os.Stderr, "- stop : cleanly powers down a running VM")
		fmt.Fprintln(os.Stderr, "- quit : quits a running VM")
		fmt.Fprintln(os.Stderr, "- status : prints status information about the ciao-down VM")
		fmt.Fprintln(os.Stderr, "- connect : connects to the VM via SSH")
		fmt.Fprintln(os.Stderr, "- delete : shuts down and deletes the VM")
	}
}

func vmFlags(fs *flag.FlagSet, memGB, CPUs *int) {
	*memGB, *CPUs = getMemAndCpus()
	fs.IntVar(memGB, "mem", *memGB, "Gigabytes of RAM allocated to VM")
	fs.IntVar(CPUs, "cpus", *CPUs, "VCPUs assignged to VM")
}

func prepareFlags() (memGB int, CPUs int, debug bool, err error) {
	fs := flag.NewFlagSet("prepare", flag.ExitOnError)
	vmFlags(fs, &memGB, &CPUs)
	fs.BoolVar(&debug, "debug", false, "Enables debug mode")
	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return -1, -1, false, err
	}

	return memGB, CPUs, debug, nil
}

func startFlags() (memGB int, CPUs int, err error) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	vmFlags(fs, &memGB, &CPUs)
	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return -1, -1, err
	}

	return memGB, CPUs, nil
}

func downloadProgress(p progress) {
	if p.totalMB >= 0 {
		fmt.Printf("Downloaded %d MB of %d\n", p.downloadedMB, p.totalMB)
	} else {
		fmt.Printf("Downloaded %d MB\n", p.downloadedMB)
	}
}

func prepare(ctx context.Context, errCh chan error) {
	if !hostSupportsNestedKVM() {
		errCh <- fmt.Errorf("Nested KVM is not enabled.  Please enable try again.")
		return
	}

	fmt.Println("Checking environment")

	memGB, CPUs, debug, err := prepareFlags()
	if err != nil {
		errCh <- err
		return
	}

	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	_, err = os.Stat(ws.instanceDir)
	if err == nil {
		errCh <- fmt.Errorf("instance already exists")
		return
	}

	fmt.Println("Installing host dependencies")
	installDeps(ctx)

	err = os.MkdirAll(ws.instanceDir, 0755)
	if err != nil {
		errCh <- fmt.Errorf("unable to create cache dir: %v", err)
		return
	}

	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(ws.instanceDir)
		}
	}()

	qcowPath, err := downloadUbuntu(ctx, ws.ciaoDir, downloadProgress)
	if err != nil {
		errCh <- err
		return
	}

	err = buildISOImage(ctx, ws.instanceDir, ws, debug)
	if err != nil {
		errCh <- err
		return
	}

	err = createRootfs(ctx, qcowPath, ws.instanceDir)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Printf("Booting VM with %d GB RAM and %d cpus\n", memGB, CPUs)

	err = bootVM(ctx, ws, memGB, CPUs)
	if err != nil {
		errCh <- err
		return
	}

	err = manageInstallation(ctx, ws.instanceDir, ws)
	if err != nil {
		errCh <- err
		return
	}
	errCh <- nil
	failed = false
	fmt.Println("VM successfully created!")
	fmt.Println("Type ciao-down connect to start using it.")
}

func start(ctx context.Context, errCh chan error) {
	if !hostSupportsNestedKVM() {
		errCh <- fmt.Errorf("Nested KVM is not enabled.  Please enable try again.")
		return
	}

	memGB, CPUs, err := startFlags()
	if err != nil {
		errCh <- err
		return
	}

	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	fmt.Printf("Booting VM with %d GB RAM and %d cpus\n", memGB, CPUs)

	err = bootVM(ctx, ws, memGB, CPUs)
	if err != nil {
		errCh <- err
		return
	}

	if err == nil {
		fmt.Println("VM Started")
	}

	errCh <- err
}

func stop(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	err = stopVM(ctx, ws.instanceDir)
	if err != nil {
		errCh <- err
		return
	}

	if err == nil {
		fmt.Println("VM Stopped")
	}

	errCh <- err
}

func quit(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	err = quitVM(ctx, ws.instanceDir)
	if err != nil {
		errCh <- err
		return
	}

	if err == nil {
		fmt.Println("VM Quit")
	}

	errCh <- err
}

func status(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	statusVM(ctx, ws.instanceDir)
	errCh <- err
}

func connect(errCh chan error) {
	path, err := exec.LookPath("ssh")
	if err != nil {
		errCh <- fmt.Errorf("Unable to locate ssh binary")
	}

	err = syscall.Exec(path, []string{path, "127.0.0.1", "-p", "10022"},
		os.Environ())
	errCh <- err
}

func delete(ctx context.Context, errCh chan error) {
	ws, err := prepareEnv(ctx)
	if err != nil {
		errCh <- err
		return
	}

	_ = quitVM(ctx, ws.instanceDir)
	err = os.RemoveAll(ws.instanceDir)
	if err != nil {
		errCh <- fmt.Errorf("unable to delete instance: %v", err)
		return
	}

	knownHosts := path.Join(ws.Home, ".ssh", "known_hosts")
	err = exec.Command("ssh-keygen", "-f", knownHosts, "-R", "[127.0.0.1]:10022").Run()
	if err != nil {
		fmt.Println("Failed to remove VM entry from known_hosts")
	}

	errCh <- nil
}

func runCommand(signalCh <-chan os.Signal) error {
	var err error

	errCh := make(chan error)
	ctx, cancelFunc := context.WithCancel(context.Background())
	switch os.Args[1] {
	case "prepare":
		go prepare(ctx, errCh)
	case "start":
		go start(ctx, errCh)
	case "stop":
		go stop(ctx, errCh)
	case "quit":
		go quit(ctx, errCh)
	case "status":
		go status(ctx, errCh)
	case "connect":
		go connect(errCh)
	case "delete":
		go delete(ctx, errCh)
	}
	select {
	case <-signalCh:
		cancelFunc()
		err = <-errCh
	case err = <-errCh:
		cancelFunc()
	}

	return err
}

func main() {
	flag.Parse()
	if len(os.Args) < 2 ||
		!(os.Args[1] == "prepare" || os.Args[1] == "start" || os.Args[1] == "stop" ||
			os.Args[1] == "quit" || os.Args[1] == "status" ||
			os.Args[1] == "connect" || os.Args[1] == "delete") {
		flag.Usage()
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := runCommand(signalCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
