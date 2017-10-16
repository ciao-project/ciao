package sdk

import (
	"fmt"
	"os"
	"text/template"
	"net/http"

	"github.com/ciao-project/ciao/openstack/image"

	"github.com/intel/tfortools"
)

func dumpImage(i *image.DefaultResponse) {
	fmt.Printf("\tName             [%s]\n", *i.Name)
	fmt.Printf("\tSize             [%d bytes]\n", i.Size)
	fmt.Printf("\tUUID             [%s]\n", i.ID)
	fmt.Printf("\tStatus           [%s]\n", i.Status)
	fmt.Printf("\tVisibility       [%s]\n", i.Visibility)
	fmt.Printf("\tTags             %v\n", i.Tags)
	fmt.Printf("\tCreatedAt        [%s]\n", i.CreatedAt)
}

func listImages() error {
	var t *template.Template
	var err error
	if Template != "" {
		t, err = tfortools.CreateTemplate("image-list", Template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	url := buildImageURL("images")
	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fatalf("Image list failed: %s", resp.Status)
	}

	var images = struct {
		Images []image.DefaultResponse
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