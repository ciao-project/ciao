package main

import (
	"fmt"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

func dumpImage(i *images.Image) {
	fmt.Printf("\tName           [%s]\n", i.Name)
	fmt.Printf("\tSize           [%d bytes]\n", i.SizeBytes)
	fmt.Printf("\tUUID           [%s]\n", i.ID)
	fmt.Printf("\tOwner          [%s]\n", i.Owner)
	fmt.Printf("\tDisk format    [%s]\n", i.DiskFormat)
	fmt.Printf("\tMinimal disk   [%d GB]\n", i.MinDiskGigabytes)
	fmt.Printf("\tMinimal memory [%d MB]\n", i.MinRAMMegabytes)
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

func createTenantImage(username, password, tenant, name string) {
	client, err := imageServiceClient(username, password, tenant)
	if err != nil {
		fatalf("Could not get Image service client [%s]\n", err)
	}

	opts := images.CreateOpts{
		Name:             name,
		DiskFormat:       "qcow2",
		MinDiskGigabytes: 8,
		MinRAMMegabytes:  512,
	}

	image, err := images.Create(client, opts).Extract()
	if err != nil {
		fatalf("Could not create image [%s]\n", err)
	}

	fmt.Printf("Created image:\n")
	dumpImage(image)
}
