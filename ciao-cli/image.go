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
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/intel/tfortools"
)

var imageCommand = &command{
	SubCommands: map[string]subCommand{
		"add":    new(imageAddCommand),
		"show":   new(imageShowCommand),
		"list":   new(imageListCommand),
		"delete": new(imageDeleteCommand),
	},
}

type imageAddCommand struct {
	Flag       flag.FlagSet
	name       string
	id         string
	file       string
	template   string
	tags       string
	visibility string
}

func (cmd *imageAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image add [flags]

Creates a new image

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", api.DefaultResponse{}, nil))
	os.Exit(2)
}

func (cmd *imageAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Image Name")
	cmd.Flag.StringVar(&cmd.id, "id", "", "Image UUID")
	cmd.Flag.StringVar(&cmd.file, "file", "", "Image file to upload")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.StringVar(&cmd.visibility, "visibility", string(api.Private),
		"Image visibility (internal,public,private)")
	cmd.Flag.StringVar(&cmd.tags, "tag", "", "Image tags (comma separated)")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func getImage(imageID string) api.DefaultResponse {
	var url string
	if checkPrivilege() && *tenantID == "admin" {
		url = buildCiaoURL("images/%s", imageID)
	} else {
		url = buildCiaoURL("%s/images/%s", *tenantID, imageID)
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.ImagesV1)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Image show failed: %s", resp.Status)
	}

	var i api.DefaultResponse

	err = unmarshalHTTPResponse(resp, &i)
	if err != nil {
		fatalf(err.Error())
	}

	return i
}

func (cmd *imageAddCommand) run(args []string) error {
	if cmd.name == "" {
		return errors.New("Missing required -name parameter")
	}

	if cmd.file == "" {
		return errors.New("Missing required -file parameter")
	}

	_, err := os.Stat(cmd.file)
	if err != nil {
		fatalf("Could not open %s [%s]\n", cmd.file, err)
	}

	imageVisibility := api.Private
	if cmd.visibility != "" {
		imageVisibility = api.Visibility(cmd.visibility)
		switch imageVisibility {
		case api.Public, api.Private, api.Internal:
		default:
			fatalf("Invalid image visibility [%v]", imageVisibility)
		}
	}

	tags := strings.Split(cmd.tags, ",")

	opts := api.CreateImageRequest{
		Name:       cmd.name,
		ID:         cmd.id,
		Visibility: imageVisibility,
		Tags:       tags,
	}

	b, err := json.Marshal(opts)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	var url string
	if checkPrivilege() && *tenantID == "admin" {
		url = buildCiaoURL("images")
	} else {
		url = buildCiaoURL("%s/images", *tenantID)
	}

	resp, err := sendCiaoRequest("POST", url, nil, body, api.ImagesV1)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		fatalf("Image creation failed: %s", resp.Status)
	}

	var image api.DefaultResponse
	err = unmarshalHTTPResponse(resp, &image)
	if err != nil {
		fatalf(err.Error())
	}

	err = uploadTenantImage(*tenantID, image.ID, cmd.file)
	if err != nil {
		fatalf(err.Error())
	}

	image = getImage(image.ID)

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "image-add", cmd.template, image, nil)
	}

	fmt.Printf("Created image:\n")
	dumpImage(&image)
	return nil
}

type imageShowCommand struct {
	Flag     flag.FlagSet
	image    string
	template string
}

func (cmd *imageShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image show

Show images
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", api.DefaultResponse{}, nil))
	os.Exit(2)
}

func (cmd *imageShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.StringVar(&cmd.image, "image", "", "Image UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageShowCommand) run(args []string) error {
	if cmd.image == "" {
		return errors.New("Missing required -image parameter")
	}

	i := getImage(cmd.image)

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "image-show", cmd.template, i, nil)
	}

	dumpImage(&i)

	return nil
}

type imageListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *imageListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] image list

List images
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s

As images are retrieved in pages, the template may be applied multiple
times.  You can not therefore rely on the length of the slice passed
to the template to determine the total number of images.
`, tfortools.GenerateUsageUndecorated([]api.DefaultResponse{}))
	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))
	os.Exit(2)
}

func (cmd *imageListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *imageListCommand) run(args []string) error {
	var t *template.Template
	var err error
	if cmd.template != "" {
		t, err = tfortools.CreateTemplate("image-list", cmd.template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	var url string
	if checkPrivilege() && *tenantID == "admin" {
		url = buildCiaoURL("images")
	} else {
		url = buildCiaoURL("%s/images", *tenantID)
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.ImagesV1)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Image list failed: %s", resp.Status)
	}

	var images = struct {
		Images []api.DefaultResponse
	}{}

	err = unmarshalHTTPResponse(resp, &images)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err = t.Execute(os.Stdout, &images.Images); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for k, i := range images.Images {
		fmt.Printf("Image #%d\n", k+1)
		dumpImage(&i)
		fmt.Printf("\n")
	}

	return err
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
	var url string
	if checkPrivilege() && *tenantID == "admin" {
		url = buildCiaoURL("images/%s", cmd.image)
	} else {
		url = buildCiaoURL("%s/images/%s", *tenantID, cmd.image)
	}

	resp, err := sendCiaoRequest("DELETE", url, nil, nil, api.ImagesV1)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Image delete failed: %s", resp.Status)
	}

	fmt.Printf("Deleted image %s\n", cmd.image)

	return nil
}

func uploadTenantImage(tenant, image, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		fatalf("Could not open %s [%s]", filename, err)
	}
	defer file.Close()

	var url string
	if checkPrivilege() && *tenantID == "admin" {
		url = buildCiaoURL("images/%s/file", image)
	} else {
		url = buildCiaoURL("%s/images/%s/file", *tenantID, image)
	}

	resp, err := sendHTTPRequestToken("PUT", url, nil, scopedToken, file, fmt.Sprintf("%s/octet-stream", api.ImagesV1))
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Unexpected HTTP response code (%d): %s", resp.StatusCode, resp.Status)
	}

	return err
}

func dumpImage(i *api.DefaultResponse) {
	fmt.Printf("\tName             [%s]\n", *i.Name)
	fmt.Printf("\tSize             [%d bytes]\n", i.Size)
	fmt.Printf("\tUUID             [%s]\n", i.ID)
	fmt.Printf("\tStatus           [%s]\n", i.Status)
	fmt.Printf("\tVisibility       [%s]\n", i.Visibility)
	fmt.Printf("\tTags             %v\n", i.Tags)
	fmt.Printf("\tCreatedAt        [%s]\n", i.CreatedAt)
}
