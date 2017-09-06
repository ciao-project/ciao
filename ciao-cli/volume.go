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

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"text/template"
	"time"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumetenants"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/intel/tfortools"
)

type customVolume volumes.Volume
type customVolumeExt volumetenants.VolumeExt
type extendedVolume struct {
	customVolumeExt
	customVolume
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var volumeCommand = &command{
	SubCommands: map[string]subCommand{
		"add":    new(volumeAddCommand),
		"list":   new(volumeListCommand),
		"show":   new(volumeShowCommand),
		"update": new(volumeUpdateCommand),
		"delete": new(volumeDeleteCommand),
		"attach": new(volumeAttachCommand),
		"detach": new(volumeDetachCommand),
	},
}

type volumeAddCommand struct {
	Flag        flag.FlagSet
	size        int
	description string
	name        string
	sourceType  string
	source      string
}

func (cmd *volumeAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume add [flags]

Create a new block storage volume

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *volumeAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Volume name")
	cmd.Flag.StringVar(&cmd.sourceType, "source_type", "image", "The type of the source to clone from")
	cmd.Flag.StringVar(&cmd.source, "source", "", "ID of image or volume to clone from")
	cmd.Flag.IntVar(&cmd.size, "size", 1, "Size of the volume in GB")
	cmd.Flag.StringVar(&cmd.description, "description", "", "Volume description")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeAddCommand) run(args []string) error {
	opts := volumes.CreateOpts{
		Description: cmd.description,
		Name:        cmd.name,
		Size:        cmd.size,
	}

	if cmd.sourceType == "image" {
		opts.ImageID = cmd.source
	} else if cmd.sourceType == "volume" {
		opts.SourceVolID = cmd.source
	} else {
		fatalf("Unknown source type [%s]\n", cmd.sourceType)
	}

	var createReq = struct {
		Volume volumes.CreateOpts
	}{
		Volume: opts,
	}

	b, err := json.Marshal(createReq)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)
	url := buildBlockURL("%s/volumes", *tenantID)
	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Volume creation failed: %s", resp.Status)
	}

	var vol = struct {
		Volume customVolume
	}{}

	err = unmarshalHTTPResponse(resp, &vol)
	if err != nil {
		fatalf(err.Error())
	}
	fmt.Printf("Created new volume: %s\n", vol.Volume.ID)

	return err
}

type volumeListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *volumeListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume list

List all volumes
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s
As volumes are retrieved in pages, the template may be applied multiple
times.  You can not therefore rely on the length of the slice passed
to the template to determine the total number of volumes.
`, tfortools.GenerateUsageUndecorated([]volumes.Volume{}))
	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))
	os.Exit(2)
}

func (cmd *volumeListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

type byName []extendedVolume

func (ss byName) Len() int      { return len(ss) }
func (ss byName) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss byName) Less(i, j int) bool {
	return ss[i].Name < ss[j].Name
}

func (cmd *volumeListCommand) run(args []string) error {
	var t *template.Template
	var err error
	if cmd.template != "" {
		t, err = tfortools.CreateTemplate("volume-list", cmd.template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	url := buildBlockURL("%s/volumes/detail", *tenantID)
	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Volume list failed: %s", resp.Status)
	}

	var vols = struct {
		Volumes []extendedVolume
	}{}

	err = unmarshalHTTPResponse(resp, &vols)
	if err != nil {
		fatalf(err.Error())
	}

	sortedVolumes := vols.Volumes
	sort.Sort(byName(sortedVolumes))

	if t != nil {
		if err = t.Execute(os.Stdout, &sortedVolumes); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for i, v := range sortedVolumes {
		fmt.Printf("Volume #%d\n", i+1)
		dumpVolume(&v.customVolume)
		fmt.Printf("\n")
	}

	return err
}

type volumeShowCommand struct {
	Flag     flag.FlagSet
	volume   string
	template string
}

func (cmd *volumeShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume show [flags]

Show information about a volume

The show flags are:
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", volumes.Volume{}, nil))
	os.Exit(2)
}

func (cmd *volumeShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.volume, "volume", "", "Volume UUID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeShowCommand) run(args []string) error {
	if cmd.volume == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	url := buildBlockURL("%s/volumes/%s", *tenantID, cmd.volume)
	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Volume show failed: %s", resp.Status)
	}

	var vol = struct {
		Volume extendedVolume
	}{}

	err = unmarshalHTTPResponse(resp, &vol)
	if err != nil {
		fatalf(err.Error())
	}

	volume := vol.Volume
	volume.customVolume.CreatedAt = volume.CreatedAt
	volume.customVolume.UpdatedAt = volume.UpdatedAt

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "volume-show", cmd.template,
			&volume, nil)
	}

	dumpVolume(&volume.customVolume)
	return nil
}

type volumeUpdateCommand struct {
	Flag        flag.FlagSet
	volume      string
	name        string
	description string
}

func (cmd *volumeUpdateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume update [flags]

Updates a volume

The update flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *volumeUpdateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.volume, "volume", "", "Volume UUID")
	cmd.Flag.StringVar(&cmd.name, "name", "", "Volume name")
	cmd.Flag.StringVar(&cmd.description, "description", "", "Volume description")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeUpdateCommand) run(args []string) error {
	if cmd.volume == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	opts := volumes.UpdateOpts{
		Name:        cmd.name,
		Description: cmd.description,
	}

	var updateReq = struct {
		Volume volumes.UpdateOpts
	}{
		Volume: opts,
	}

	b, err := json.Marshal(updateReq)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)
	url := buildBlockURL("%s/volumes/%s", *tenantID, cmd.volume)
	resp, err := sendHTTPRequest("PUT", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Volume update failed: %s", resp.Status)
	}

	var vol = struct {
		Volume customVolume
	}{}

	err = unmarshalHTTPResponse(resp, &vol)
	if err != nil {
		fatalf(err.Error())
	}

	if err == nil {
		fmt.Printf("Updated volume: %s\n", vol.Volume.ID)
	}

	return err
}

type volumeDeleteCommand struct {
	Flag   flag.FlagSet
	volume string
}

func (cmd *volumeDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume delete [flags]

Deletes a volume

The delete flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *volumeDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.volume, "volume", "", "Volume UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeDeleteCommand) run(args []string) error {
	if cmd.volume == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	url := buildBlockURL("%s/volumes/%s", *tenantID, cmd.volume)
	resp, err := sendHTTPRequest("DELETE", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Volume delete failed: %s", resp.Status)
	}

	return err
}

type volumeAttachCommand struct {
	Flag       flag.FlagSet
	volume     string
	instance   string
	mountpoint string
	mode       string
}

func (cmd *volumeAttachCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume attach [flags]

Attachs a volume to an instance

The attach flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *volumeAttachCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.volume, "volume", "", "Volume UUID")
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.StringVar(&cmd.mountpoint, "mountpoint", "/mnt", "Mount point")
	cmd.Flag.StringVar(&cmd.mountpoint, "mode", "rw", "Access mode")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeAttachCommand) run(args []string) error {
	if cmd.volume == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	if cmd.instance == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	// mountpoint or mode isn't required

	options := volumeactions.AttachOpts{
		MountPoint:   cmd.mountpoint,
		Mode:         volumeactions.AttachMode(cmd.mode),
		InstanceUUID: cmd.instance,
	}

	var attachReq = struct {
		Attach volumeactions.AttachOpts `json:"os-attach"`
	}{
		Attach: options,
	}

	b, err := json.Marshal(attachReq)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)
	url := buildBlockURL("%s/volumes/%s/action", *tenantID, cmd.volume)
	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Volume attach failed: %s", resp.Status)
	}

	if err == nil {
		fmt.Printf("Attached volume: %s\n", cmd.volume)
	}
	return err
}

type volumeDetachCommand struct {
	Flag   flag.FlagSet
	volume string
}

func (cmd *volumeDetachCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] volume detach [flags]

Detaches a volume from an instance

The detach flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *volumeDetachCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.volume, "volume", "", "Volume UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *volumeDetachCommand) run(args []string) error {
	if cmd.volume == "" {
		errorf("missing required -volume parameter")
		cmd.usage()
	}

	// mountpoint or mode isn't required

	var detachReq = struct {
		Detach volumeactions.DetachOpts `json:"os-detach"`
	}{
		Detach: volumeactions.DetachOpts{},
	}

	b, err := json.Marshal(detachReq)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)
	url := buildBlockURL("%s/volumes/%s/action", *tenantID, cmd.volume)
	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Volume detach failed: %s", resp.Status)
	}

	if err == nil {
		fmt.Printf("Detached volume: %s\n", cmd.volume)
	}
	return err
}

func dumpVolume(v *customVolume) {
	fmt.Printf("\tName             [%s]\n", v.Name)
	fmt.Printf("\tSize             [%d GB]\n", v.Size)
	fmt.Printf("\tUUID             [%s]\n", v.ID)
	fmt.Printf("\tStatus           [%s]\n", v.Status)
	fmt.Printf("\tDescription      [%s]\n", v.Description)
}
