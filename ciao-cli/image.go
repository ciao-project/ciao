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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

type imageDiskFormat string

func (f *imageDiskFormat) Set(value string) error {
	*f = imageDiskFormat(value)
	supportedFormats := []string{"ami", "ari", "aki", "vhd", "vmdk", "raw", "qcow2", "vdi", "iso"}
	for _, format := range supportedFormats {
		if format == value {
			return nil
		}
	}
	return fmt.Errorf("Supported image disk formats are: %s", supportedFormats)
}

func (f *imageDiskFormat) String() string {
	return fmt.Sprint(*f)
}

type imageContainerFormat string

func (f *imageContainerFormat) Set(value string) error {
	*f = imageContainerFormat(value)
	supportedFormats := []string{"ami", "ari", "aki", "bare", "ovf", "ova", "docker"}
	for _, format := range supportedFormats {
		if format == value {
			return nil
		}
	}
	return fmt.Errorf("Supported image container formats are: %s", supportedFormats)
}

func (f *imageContainerFormat) String() string {
	return fmt.Sprint(*f)
}

var imageCommand = &command{
	SubCommands: map[string]subCommand{
		"add":      new(imageAddCommand),
		"list":     new(imageListCommand),
		"upload":   new(imageUploadCommand),
		"download": new(imageDownloadCommand),
		"delete":   new(imageDeleteCommand),
		"modify":   new(imageModifyCommand),
	},
}

type imageAddCommand struct {
	Flag            flag.FlagSet
	name            string
	containerFormat imageContainerFormat
	diskFormat      imageDiskFormat
	minDiskSize     int
	minRAMSize      int
	visibility      string
	protected       bool
	tags            string
	file            string
}

func (cmd *imageAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image add [flags]

Creates a new image

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageAddCommand) parseArgs(args []string) []string {
	cmd.containerFormat = "bare"
	cmd.diskFormat = "qcow2"
	cmd.Flag.StringVar(&cmd.name, "name", "", "Image Name")
	cmd.Flag.Var(&cmd.containerFormat, "container-format", "Image Container Format (ami, ari, aki, bare, ovf, ova, docker")
	cmd.Flag.Var(&cmd.diskFormat, "disk-format", "Image Disk Format (ami, ari, aki, vhd, vmdk, raw, qcow2, vdi, iso")
	cmd.Flag.IntVar(&cmd.minDiskSize, "min-disk-size", 0, "Minimum disk size in GB")
	cmd.Flag.IntVar(&cmd.minRAMSize, "min-ram-size", 0, "Minimum amount of RAM in MB")
	cmd.Flag.StringVar(&cmd.visibility, "visibility", "public", "Image visibility (public or private)")
	cmd.Flag.BoolVar(&cmd.protected, "protected", false, "Prevent an image from being deleted")
	cmd.Flag.StringVar(&cmd.tags, "tags", "", "Image tags separated by comma")
	cmd.Flag.StringVar(&cmd.file, "file", "", "Image file to upload")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageAddCommand) run(args []string) error {
	if cmd.name == "" {
		return errors.New("Missing required -name parameter")
	}

	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	var visibility images.ImageVisibility
	if cmd.visibility == "public" {
		visibility = images.ImageVisibilityPublic
	} else if cmd.visibility == "private" {
		visibility = images.ImageVisibilityPrivate
	} else {
		fatalf("Image visibility should be public or private")
	}

	opts := images.CreateOpts{
		Name:             cmd.name,
		ContainerFormat:  cmd.containerFormat.String(),
		DiskFormat:       cmd.diskFormat.String(),
		MinDiskGigabytes: cmd.minDiskSize,
		MinRAMMegabytes:  cmd.minRAMSize,
		Visibility:       &visibility,
		Protected:        cmd.protected,
		Tags:             strings.Split(cmd.tags, ","),
	}

	image, err := images.Create(client, opts).Extract()
	if err != nil {
		fatalf("Could not create image [%s]\n", err)
	}

	if cmd.file != "" {
		uploadTenantImage(*identityUser, *identityPassword, *tenantID, image.ID, cmd.file)
		image, err = images.Get(client, image.ID).Extract()
		if err != nil {
			fatalf("Could not retrieve new created image [%s]\n", err)
		}
	}

	fmt.Printf("Created image:\n")
	dumpImage(image)
	return nil
}

type imageListCommand struct {
	Flag flag.FlagSet
}

func (cmd *imageListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image list

List images
`)
	os.Exit(2)
}

func (cmd *imageListCommand) parseArgs(args []string) []string {
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageListCommand) run(args []string) error {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	pager := images.List(client, images.ListOpts{})

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		imageList, err := images.ExtractImages(page)
		if err != nil {
			errorf("Could not extract image [%s]\n", err)
		}
		for k, i := range imageList {
			fmt.Printf("Image #%d\n", k+1)
			dumpImage(&i)
			fmt.Printf("\n")
		}
		return false, nil
	})
	return err
}

type imageUploadCommand struct {
	Flag  flag.FlagSet
	image string
	file  string
}

func (cmd *imageUploadCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image upload [flags]

Uploads a file to an image data

The upload flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageUploadCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.StringVar(&cmd.file, "file", "", "File to upload")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageUploadCommand) run(args []string) error {
	return uploadTenantImage(*identityUser, *identityPassword, *tenantID, cmd.image, cmd.file)
}

type imageDownloadCommand struct {
	Flag  flag.FlagSet
	image string
	file  string
}

func (cmd *imageDownloadCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image download [flags]

Fetch an image

The download flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageDownloadCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.StringVar(&cmd.file, "file", "", "Filename to save the image (default will print to stdout)")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageDownloadCommand) run(args []string) error {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	r, err := images.Download(client, cmd.image).Extract()
	if err != nil {
		fatalf("Could not download image [%s]\n", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		fatalf("Could not read [%s]\n", err)
	}

	if cmd.file == "" {
		fmt.Printf("%s\n", b)
	} else {
		err := ioutil.WriteFile(cmd.file, b, 0644)
		if err != nil {
			fatalf("Could not write image to file [%s]\n", err)
		}
	}
	return nil
}

type imageDeleteCommand struct {
	Flag  flag.FlagSet
	image string
}

func (cmd *imageDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image delete [flags]

Deletes an image

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageDeleteCommand) run(args []string) error {
	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	res := images.Delete(client, cmd.image)
	if res.Err != nil {
		fatalf("Could not delete Image [%s]\n", res.Err)
	}
	fmt.Printf("Deleted image %s\n", cmd.image)
	return res.Err
}

func uploadTenantImage(username, password, tenant, image, filename string) error {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		fatalf("Could not open %s [%s]", filename, err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)

	file.Read(buffer)
	fileBytes := bytes.NewReader(buffer)

	res := images.Upload(client, image, fileBytes)
	if res.Err != nil {
		fatalf("Could not upload %s [%s]", filename, res.Err)
	}
	return res.Err
}

type imageModifyCommand struct {
	Flag       flag.FlagSet
	name       string
	visibility string
	tags       string
	image      string
}

func (cmd *imageModifyCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image modify [flags]

Modify an image

The modify flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *imageModifyCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Image Name")
	cmd.Flag.StringVar(&cmd.visibility, "visibility", "public", "Image visibility (public or private)")
	cmd.Flag.StringVar(&cmd.tags, "tags", "", "Image tags separated by comma")
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageModifyCommand) run(args []string) error {
	if cmd.image == "" {
		return errors.New("Missing required -image parameter")
	}

	client, err := imageServiceClient(*identityUser, *identityPassword, *tenantID)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	var opts images.UpdateOpts
	if cmd.visibility != "" {
		var visibility images.ImageVisibility
		if cmd.visibility == "public" {
			visibility = images.ImageVisibilityPublic
		} else if cmd.visibility == "private" {
			visibility = images.ImageVisibilityPrivate
		} else {
			fatalf("Image visibility should be public or private")
		}
		v := images.UpdateVisibility{
			Visibility: visibility,
		}
		opts = append(opts, v)
	}

	if cmd.name != "" {
		n := images.ReplaceImageName{
			NewName: cmd.name,
		}
		opts = append(opts, n)
	}

	if len(cmd.tags) > 0 {
		inputTags := strings.Split(cmd.tags, ",")
		t := images.ReplaceImageTags{
			NewTags: inputTags,
		}
		opts = append(opts, t)
	}

	image, err := images.Update(client, cmd.image, opts).Extract()
	if err != nil {
		fatalf("Could not update image's properties [%s]\n", err)
	}

	fmt.Printf("Updated image:\n")
	dumpImage(image)
	return nil
}

func dumpImage(i *images.Image) {
	fmt.Printf("\tName             [%s]\n", i.Name)
	fmt.Printf("\tSize             [%d bytes]\n", i.SizeBytes)
	fmt.Printf("\tUUID             [%s]\n", i.ID)
	fmt.Printf("\tStatus           [%s]\n", i.Status)
	fmt.Printf("\tOwner            [%s]\n", i.Owner)
	fmt.Printf("\tDisk format      [%s]\n", i.DiskFormat)
	fmt.Printf("\tContainer format [%s]\n", i.ContainerFormat)
	fmt.Printf("\tMinimal disk     [%d GB]\n", i.MinDiskGigabytes)
	fmt.Printf("\tMinimal memory   [%d MB]\n", i.MinRAMMegabytes)
	fmt.Printf("\tChecksum         [%s]\n", i.Checksum)
	fmt.Printf("\tVisibility       [%s]\n", i.Visibility)
	fmt.Printf("\tProtected        [%t]\n", i.Protected)
	fmt.Printf("\tCreatedDate      [%s]\n", i.CreatedDate)
	fmt.Printf("\tLastUpdate       [%s]\n", i.LastUpdate)
	fmt.Printf("\tTags             [%s]\n", i.Tags)
	fmt.Printf("\tFile             [%s]\n", i.File)
	fmt.Printf("\tSchema           [%s]\n", i.Schema)
}

func imageServiceClient(username, password, tenant string) (*gophercloud.ServiceClient, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: *identityURL + "/v3/",
		Username:         username,
		Password:         password,
		DomainID:         "default",
		TenantID:         tenant,
		AllowReauth:      true,
	}

	provider, err := newAuthenticatedClient(opt)
	if err != nil {
		errorf("Could not get AuthenticatedClient %s\n", err)
		return nil, err
	}

	return openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Name:   "glance",
		Region: "RegionOne",
	})
}
