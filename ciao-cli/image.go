package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

type imageVisibility string

func (v *imageVisibility) Set(value string) error {
	*v = imageVisibility(value)
	if value == "public" || value == "private" {
		return nil
	}
	return fmt.Errorf("visibility should be either private or public")
}

func (v *imageVisibility) String() string {
	return fmt.Sprint(*v)
}

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

	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		errorf("Could not get AuthenticatedClient %s\n", err)
	}

	return openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Name:   "glance",
		Region: "RegionOne",
	})
}

func listTenantImages(username, password, tenant string) {
	client, err := imageServiceClient(username, password, tenant)
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
}

func createTenantImage(username, password, tenant, filePath string, opts *images.CreateOpts) {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	image, err := images.Create(client, opts).Extract()
	if err != nil {
		fatalf("Could not create image [%s]\n", err)
	}

	if filePath != "" {
		uploadTenantImage(username, password, tenant, image.ID, filePath)
		image, _ = images.Get(client, image.ID).Extract()
	}

	fmt.Printf("Created image:\n")
	dumpImage(image)
}

func deleteTenantImage(username, password, tenant, imageID string) {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	res := images.Delete(client, imageID)
	if res.Err != nil {
		fatalf("Could not delete Image [%s]\n", res.Err)
	}
}

func uploadTenantImage(username, password, tenant, imageID, filePath string) {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		fatalf("Could not open %s [%s]", filePath, err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)

	file.Read(buffer)
	fileBytes := bytes.NewReader(buffer)

	res := images.Upload(client, imageID, fileBytes)
	if res.Err != nil {
		fatalf("Could not upload %s [%s]", filePath, res.Err)
	}
}

func downloadTenantImage(username, password, tenant, imageID, filePath string) {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	r, err := images.Download(client, imageID).Extract()
	if err != nil {
		fatalf("Could not download image [%s]\n", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		fatalf("Could not read [%s]\n", err)
	}

	if filePath == "" {
		fmt.Printf("%s\n", b)
	} else {
		err := ioutil.WriteFile(filePath, b, 0644)
		if err != nil {
			fatalf("Could not write image to file [%s]\n", err)
		}
	}
}
