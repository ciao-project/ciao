// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"io/ioutil"
	"os"
	"regexp"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/uuid"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create cluster objects",
}

var imgFlags = struct {
	id         string
	visibility string
}{}

var instanceFlags = struct {
	instances int
	label     string
	name      string
	workload  string
}{}

var tenantFlags = struct {
	cidrPrefixSize             int
	name                       string
	createPrivilegedContainers bool
}{}

var volFlags = struct {
	description string
	name        string
	size        int
	source      string
	sourcetype  string
}{}

var imageCreateCmd = &cobra.Command{
	Use:   "image NAME FILE",
	Short: `Add an image to the cluster`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		file := args[1]

		f, err := os.Open(file)
		if err != nil {
			return errors.Wrap(err, "Error opening image file")
		}
		defer func() { _ = f.Close() }()

		imageVisibility := types.Private
		if imgFlags.visibility != "" {
			imageVisibility = types.Visibility(imgFlags.visibility)
			switch imageVisibility {
			case types.Public, types.Private, types.Internal:
			default:
				return errors.New("Invalid image visibility")
			}
		}

		id, err := c.CreateImage(name, imageVisibility, imgFlags.id, f)
		if err != nil {
			return errors.Wrap(err, "Error creating image")
		}

		image, err := c.GetImage(id)
		if err != nil {
			return errors.Wrap(err, "Errror getting image")
		}

		return render(cmd, image)
	},
}

func validateCreateCommandArgs() error {
	if instanceFlags.instances < 1 {
		return errors.New("Invalid instance count")
	}

	if instanceFlags.name != "" {
		r := regexp.MustCompile("^[a-z0-9-]{1,64}?$")
		if !r.MatchString(instanceFlags.name) {
			return errors.New("Requested name must be between 1 and 64 lowercase letters, numbers and hyphens")
		}
	}

	return nil
}

func populateCreateServerRequest(server *api.CreateServerRequest) {
	if instanceFlags.label != "" {
		server.Server.Metadata = make(map[string]string)
		server.Server.Metadata["label"] = instanceFlags.label
	}

	server.Server.MaxInstances = instanceFlags.instances
	server.Server.MinInstances = 1
	server.Server.Name = instanceFlags.name
}

var instanceCreateCmd = &cobra.Command{
	Use:   "instance WORKLOAD",
	Short: "Create an instance of a workload",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCreateCommandArgs(); err != nil {
			return err
		}

		var server api.CreateServerRequest

		server.Server.WorkloadID = args[0]

		populateCreateServerRequest(&server)

		servers, err := c.CreateInstances(server)
		if err != nil {
			return errors.Wrap(err, "Error creating instances")
		}

		return render(cmd, servers.Servers)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var poolCreateCmd = &cobra.Command{
	Use:   "pool NAME",
	Short: `Add a pool to the cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.CreateExternalIPPool(args[0]), "Error creating external IP pool")
	},
}

var tenantCreateCmd = &cobra.Command{
	Use:   "tenant ID",
	Short: "Create a new tenant in the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Creating tenants is restricted to privileged users")
		}

		tenantID := args[0]

		// CIDR prefix size must be between 12 and 30 bits
		if tenantFlags.cidrPrefixSize != 0 && (tenantFlags.cidrPrefixSize > 30 || tenantFlags.cidrPrefixSize < 12) {
			return errors.New("Subnet prefix must be 12-30")
		}

		tuuid, err := uuid.Parse(tenantID)
		if err != nil {
			return errors.New("Tenant ID must be a UUID")
		}

		config := types.TenantConfig{
			Name:       tenantFlags.name,
			SubnetBits: tenantFlags.cidrPrefixSize,
		}
		config.Permissions.PrivilegedContainers = tenantFlags.createPrivilegedContainers

		summary, err := c.CreateTenantConfig(tuuid.String(), config)
		if err != nil {
			return errors.Wrap(err, "Error creating tenant")
		}

		return render(cmd, summary)
	},
}

var volumeCreateCmd = &cobra.Command{
	Use:   "volume",
	Short: "Create a volume in the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		createReq := api.RequestedVolume{
			Description: volFlags.description,
			Name:        volFlags.name,
			Size:        volFlags.size,
		}

		if volFlags.sourcetype == "image" {
			createReq.ImageRef = volFlags.source
		} else if volFlags.sourcetype == "volume" {
			createReq.SourceVolID = volFlags.source
		}

		vol, err := c.CreateVolume(createReq)
		if err != nil {
			return errors.Wrap(err, "Error creating volume")
		}

		return render(cmd, vol)
	},
}

type source struct {
	Type   types.SourceType `yaml:"type"`
	Source string           `yaml:"source"`
}

type disk struct {
	ID        *string `yaml:"volume_id,omitempty"`
	Size      int     `yaml:"size"`
	Bootable  bool    `yaml:"bootable"`
	Source    source  `yaml:"source"`
	Ephemeral bool    `yaml:"ephemeral"`
}

type workloadRequirements struct {
	VCPUs      int    `yaml:"vcpus"`
	MemMB      int    `yaml:"mem_mb"`
	NodeID     string `yaml:"node_id,omitempty"`
	Hostname   string `yaml:"hostname,omitempty"`
	Privileged bool   `yaml:"privileged,omitempty"`
}

type workloadOptions struct {
	Description     string               `yaml:"description"`
	VMType          string               `yaml:"vm_type"`
	FWType          string               `yaml:"fw_type,omitempty"`
	ImageName       string               `yaml:"image_name,omitempty"`
	Requirements    workloadRequirements `yaml:"requirements"`
	CloudConfigFile string               `yaml:"cloud_init,omitempty"`
	Disks           []disk               `yaml:"disks,omitempty"`
}

func optToReqStorage(opt workloadOptions) ([]types.StorageResource, error) {
	storage := make([]types.StorageResource, 0)
	bootableCount := 0
	for _, disk := range opt.Disks {
		res := types.StorageResource{
			Size:      disk.Size,
			Bootable:  disk.Bootable,
			Ephemeral: disk.Ephemeral,
		}

		// Use existing volume
		if disk.ID != nil {
			res.ID = *disk.ID
		} else {
			// Create a new one
			if disk.Source.Type == "" {
				disk.Source.Type = types.Empty
			}

			if disk.Source.Type != types.Empty {
				res.SourceType = disk.Source.Type
				res.Source = disk.Source.Source

				if res.Source == "" {
					return nil, errors.New("Invalid workload yaml: when using a source an id must also be specified")
				}
			} else {
				if disk.Bootable == true {
					// you may not request a bootable drive
					// from an empty source
					return nil, errors.New("Invalid workload yaml: empty disk source may not be bootable")
				}

				if disk.Size <= 0 {
					return nil, errors.New("Invalid workload yaml: size required when creating a volume")
				}
			}
		}

		if disk.Bootable {
			bootableCount++
		}

		storage = append(storage, res)
	}

	if payloads.Hypervisor(opt.VMType) == payloads.QEMU && bootableCount == 0 {
		return nil, errors.New("Invalid workload yaml: no bootable disks specified for a VM")
	}

	return storage, nil
}

func optToReq(opt workloadOptions, req *types.Workload) error {
	b, err := ioutil.ReadFile(opt.CloudConfigFile)
	if err != nil {
		return err
	}

	config := string(b)

	// this is where you'd validate that the options make
	// sense.
	req.Description = opt.Description
	req.VMType = payloads.Hypervisor(opt.VMType)
	req.FWType = opt.FWType
	req.ImageName = opt.ImageName
	req.Config = config
	req.Storage, err = optToReqStorage(opt)

	if err != nil {
		return err
	}

	req.Requirements.MemMB = opt.Requirements.MemMB
	req.Requirements.VCPUs = opt.Requirements.VCPUs
	req.Requirements.Hostname = opt.Requirements.Hostname
	req.Requirements.NodeID = opt.Requirements.NodeID
	req.Requirements.Privileged = opt.Requirements.Privileged

	return nil
}

var workloadCreateCmd = &cobra.Command{
	Use:   "workload FILE",
	Short: `Create a new workload`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var opt workloadOptions
		var req types.Workload

		config := args[0]

		f, err := ioutil.ReadFile(config)
		if err != nil {
			return errors.Wrap(err, "Error reading config file")
		}

		err = yaml.Unmarshal(f, &opt)
		if err != nil {
			return errors.Wrap(err, "Error unmarshalling file")
		}

		err = optToReq(opt, &req)
		if err != nil {
			return errors.Wrap(err, "Error converting options to request")
		}

		workload, err := c.CreateWorkload(req)
		if err != nil {
			return errors.Wrap(err, "Error creating workload")
		}

		return render(cmd, workload)
	},
}

var createCmds = []*cobra.Command{imageCreateCmd, instanceCreateCmd, poolCreateCmd, volumeCreateCmd, workloadCreateCmd, tenantCreateCmd}

func init() {
	for _, cmd := range createCmds {
		createCmd.AddCommand(cmd)
	}
	rootCmd.AddCommand(createCmd)

	imageCreateCmd.Flags().StringVar(&imgFlags.id, "id", "", "Image ID")
	imageCreateCmd.Flags().StringVar(&imgFlags.visibility, "visibility", "private", "Image visibility (internal,public,private)")

	instanceCreateCmd.Flags().IntVar(&instanceFlags.instances, "instances", 1, "Number of instances to create")
	instanceCreateCmd.Flags().StringVar(&instanceFlags.label, "label", "", "Set a frame label. This will trigger frame tracing")
	instanceCreateCmd.Flags().StringVar(&instanceFlags.name, "name", "", "Name for this instance. When multiple instances are requested this is used as a prefix")
	instanceCreateCmd.Flags().StringVar(&instanceFlags.workload, "workload", "", "Workload UUID")

	volumeCreateCmd.Flags().StringVar(&volFlags.description, "description", "", "Volume description")
	volumeCreateCmd.Flags().StringVar(&volFlags.name, "name", "", "Volume name")
	volumeCreateCmd.Flags().IntVar(&volFlags.size, "size", 1, "Size of the volume in GiB")
	volumeCreateCmd.Flags().StringVar(&volFlags.source, "source", "", "ID of image or volume to clone from")
	volumeCreateCmd.Flags().StringVar(&volFlags.sourcetype, "source-type", "image", "The type of the source to clone from")

	tenantCreateCmd.Flags().IntVar(&tenantFlags.cidrPrefixSize, "cidr-prefix-size", 0, "Number of bits in network mask (12-30)")
	tenantCreateCmd.Flags().BoolVar(&tenantFlags.createPrivilegedContainers, "create-privileged-containers", false, "Whether this tenant can create privileged containers")
	tenantCreateCmd.Flags().StringVar(&tenantFlags.name, "name", "", "Tenant name")
}
