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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/ciao-project/ciao/ciao/tool"
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/uuid"

	"github.com/intel/tfortools"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

)

var config string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add various ciao objects",
	Long:  `Add the objects below to the ciao cluster`,
	Args:  cobra.MinimumNArgs(2),
}

type imageAddFlags struct {
	name       string
	id         string
	file       string
	visibility string
}

type volumeFlag struct {
	uuid      string
	bootIndex string
	swap      bool
	local     bool
	ephemeral bool
	size      int
	tag       string
}
type volumeFlagSlice []volumeFlag

type instanceAddFlags struct {
	instances	int
	label		string
	name		string
	volumes		volumeFlagSlice
	workload	string	
}

type poolAddFlags struct {
	name		string
	subnet		string
	new			bool
}

type tenantAddFlags struct {
	cidrPrefixSize	int
	name			string
	tenantID		string
}

type volumeAddFlags struct {
	description	string
	name		string
	size		int
	source		string
	sourcetype	string
}

var imgFlags imageAddFlags
var instanceFlags instanceAddFlags
var poolFlags poolAddFlags
var tenantFlags tenantAddFlags
var volFlags volumeAddFlags

var imageAddCmd = &cobra.Command{
	Use:  "image",
	Long: `Add a specific image to the ciao cluster.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if imgFlags.name == "" {
			fmt.Println("Missing required --name parameter")
			return
		}

		if imgFlags.file == "" {
			fmt.Println("Missing required --file parameter")
			return
		}

		f, err := os.Open(imgFlags.file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open %s [%s]\n", imgFlags.file, err)
			return
		}
		defer func() { _ = f.Close() }()

		imageVisibility := types.Private
		if imgFlags.visibility != "" {
			imageVisibility = types.Visibility(imgFlags.visibility)
			switch imageVisibility {
			case types.Public, types.Private, types.Internal:
			default:
				fmt.Fprintf(os.Stderr, "Invalid image visibility [%v]", imageVisibility)
				return
			}
		}

		id, err := C.CreateImage(imgFlags.name, imageVisibility, imgFlags.id, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating image: %s\n", err)
			return
		}

		image, err := C.GetImage(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting image: %s\n", err)
			return
		}

		if C.Template != "" {
			tfortools.OutputToTemplate(os.Stdout, "image-add", C.Template, image, nil)
			return
		}

		fmt.Printf("Created image\n")
	},
}

func validateAddCommandArgs(cmd instanceAddFlags) {
	if C.TenantID == "" {
		fmt.Println("Missing required --tenant-id parameter")
		return
	}

	if cmd.workload == "" {
		fmt.Println("Missing required --workload parameter")
		return
	}

	if cmd.instances < 1 {
		fmt.Fprintf(os.Stderr, "Invalid value for -instances: %d", cmd.instances)
		return
	}

	if cmd.name != "" {
		r := regexp.MustCompile("^[a-z0-9-]{1,64}?$")
		if !r.MatchString(cmd.name) {
			fmt.Println("Requested name must be between 1 and 64 lowercase letters, numbers and hyphens")
		}
	}

	for _, volume := range cmd.volumes {
		//NOTE: volume.uuid itself may only be validated by controller as
		//only it knows which storage interface is in use and what
		//constitutes a valid uuid for that storage implementation

		if cmd.instances != 1 && volume.uuid != "" {
			fmt.Fprintf(os.Stderr, "Cannot attach volume by uuid (\"-volume uuid=%s\") to multiple instances (\"-instances=%d\")",
				volume.uuid, cmd.instances)
			return
		}
	}
}
func validateCreateServerRequest(server api.CreateServerRequest) error {
	for _, bd := range server.Server.BlockDeviceMappings {
		if bd.DestinationType == "local" && bd.UUID != "" {
			return fmt.Errorf("Only one of \"uuid={UUID}\" or \"local\" sub-arguments may be specified")
		}

		if bd.VolumeSize != 0 && bd.UUID != "" {
			return fmt.Errorf("Only one of \"uuid={UUID}\" or \"size={SIZE}\" sub-arguments may be specificed")
		}
	}

	return nil
}

func populateCreateServerRequest(cmd instanceAddFlags, server *api.CreateServerRequest) {
	if cmd.label != "" {
		server.Server.Metadata = make(map[string]string)
		server.Server.Metadata["label"] = cmd.label
	}

	server.Server.WorkloadID = cmd.workload
	server.Server.MaxInstances = cmd.instances
	server.Server.MinInstances = 1
	server.Server.Name = cmd.name

	for _, volume := range cmd.volumes {
		bd := api.BlockDeviceMapping{
			DeviceName:          "", //unsupported
			DeleteOnTermination: volume.ephemeral,
			BootIndex:           volume.bootIndex,
			Tag:                 volume.tag,
			UUID:                volume.uuid,
			VolumeSize:          volume.size,
		}

		if volume.local {
			bd.DestinationType = "local"
		} else {
			bd.DestinationType = "volume"
		}

		if bd.DestinationType == "volume" && volume.uuid != "" {
			// treat all uuid specified items as
			// volumes, ciao internals will figure out
			// if it is an image, volume or snapshot
			bd.SourceType = "volume"
		} else {
			bd.SourceType = "blank"
		}

		if volume.swap {
			bd.GuestFormat = "swap"
		} else {
			bd.GuestFormat = "ephemeral"
		}

		server.Server.BlockDeviceMappings = append(server.Server.BlockDeviceMappings, bd)
	}
}

// implement the flag.Value interface, eg:
// type Value interface {
// 	String() string
// 	Set(string) error
// }
func (v *volumeFlagSlice) String() string {
	var out string

	for _, vol := range *v {
		var subArgs []string
		if vol.uuid != "" {
			subArgs = append(subArgs, "uuid="+vol.uuid)
		}
		if vol.bootIndex != "" && vol.bootIndex != "none" {
			subArgs = append(subArgs, "boot_index="+vol.bootIndex)
		}
		if vol.swap {
			subArgs = append(subArgs, "swap")
		}
		if vol.local {
			subArgs = append(subArgs, "local")
		}
		if vol.ephemeral {
			subArgs = append(subArgs, "ephemeral")
		}
		if vol.size != 0 {
			subArgs = append(subArgs, "size="+fmt.Sprintf("%d", vol.size))
		}
		if vol.tag != "" {
			subArgs = append(subArgs, "tag="+vol.tag)
		}

		out += "-volume "
		subArgCount := len(subArgs)
		for subArgIdx, subArg := range subArgs {
			out += subArg
			if subArgIdx < subArgCount-1 {
				out += ","
			}
		}

		out += "\n"
	}

	return out
}
func (v *volumeFlagSlice) Set(value string) error {
	if value == "" {
		return fmt.Errorf("Invalid empty volume argument list")
	}

	stringArgsMap := make(map[string]string)
	boolArgsMap := make(map[string]bool)
	intArgsMap := make(map[string]int)

	subArgs := strings.Split(value, ",")
	for _, subArg := range subArgs {
		if subArg == "" {
			continue
		}
		err := processInstanceVolumeSubArg(subArg, stringArgsMap, boolArgsMap, intArgsMap)
		if err != nil {
			return err
		}
	}

	vol := volumeFlag{
		uuid:      stringArgsMap["uuid"],
		bootIndex: stringArgsMap["boot_index"],
		swap:      boolArgsMap["swap"],
		local:     boolArgsMap["local"],
		ephemeral: boolArgsMap["ephemeral"],
		size:      intArgsMap["size"],
		tag:       stringArgsMap["tag"],
	}
	*v = append(*v, vol)

	err := validateInstanceVolumeSubArgCombo(v)
	if err != nil {
		return err
	}

	return nil
}
func (v *volumeFlagSlice) Type() string {
	return string("[]volumeFlag")
}

func argMatch(patterns []string, arg string) bool {
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, arg)
		if matched {
			return true
		}
	}
	return false
}
func isInstanceVolumeBoolArg(arg string) bool {
	if isInstanceVolumeImplicitBoolArg(arg) {
		return true
	}

	return isInstanceVolumeExplicitBoolArg(arg)
}

func isInstanceVolumeImplicitBoolArg(arg string) bool {
	patterns := []string{
		"^swap$",
		"^local$",
		"^ephemeral$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeExplicitBoolArg(arg string) bool {
	patterns := []string{
		"^swap=(true|false)$",
		"^local=(true|false)$",
		"^ephemeral=(true|false)$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeIntArg(arg string) bool {
	patterns := []string{
		"^size=[0-9]+$",
	}
	return argMatch(patterns, arg)
}
func isInstanceVolumeStringArg(arg string) bool {
	patterns := []string{
		"^uuid=.*$",
		"^boot_index=.*$",
		"^tag=.*$",
	}
	return argMatch(patterns, arg)
}
func getInstanceVolumeImplicitBoolArgs(subArg string, boolArgsMap map[string]bool) bool {
	// search for implicit affirmative bools by name, put in map
	if !isInstanceVolumeImplicitBoolArg(subArg) {
		return false
	}

	boolArgsMap[subArg] = true
	return true
}
func getInstanceVolumeExplicitBoolArgs(key string, val string, boolArgsMap map[string]bool) (bool, error) {
	// search for explicit bools by name, put in map
	if !isInstanceVolumeImplicitBoolArg(key) {
		return false, nil
	}
	fullArg := key + "=" + val
	if !isInstanceVolumeExplicitBoolArg(fullArg) {
		return false, fmt.Errorf("Invalid argument. Expected %s={true|false}, got \"%s=%s\"",
			key, key, val)
	}

	if boolArgsMap[key] != false {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%t\", got additional  \"%s=%s\"",
			key, boolArgsMap[key], key, val)
	}

	if val == "true" {
		boolArgsMap[key] = true
		return true, nil
	} else if val == "false" {
		boolArgsMap[key] = false
		return true, nil
	}

	return false, fmt.Errorf("Invalid argument. Expected %s={true|false}, got \"%s=%s\"",
		key, key, val)
}
func getInstanceVolumeIntegerArgs(key string, val string, intArgsMap map[string]int) (bool, error) {
	// search for integer args by name, put in map
	if key != "size" {
		return false, nil
	}

	if intArgsMap[key] != 0 {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%d\", got additional \"%s=%s\"",
			key, intArgsMap[key], key, val)
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return false, fmt.Errorf("Invalid argument. Expected %s={integer}, got \"%s=%s\": %s", key, key, val, err)
	}

	intArgsMap[key] = i
	return true, nil
}
func getInstanceVolumeStringArgs(key string, val string, stringArgsMap map[string]string) (bool, error) {
	if stringArgsMap[key] != "" {
		return false, fmt.Errorf("Conflicting arguments. Already had \"%s=%s\", got additional \"%s=%s\"",
			key, stringArgsMap[key], key, val)
	}

	if val == "" {
		return false, fmt.Errorf("Invalid argument. Expected %s={string}, got \"%s=%s\"", key, key, val)
	}

	if key == "boot_index" {
		goodIndex := false
		if val == "none" {
			goodIndex = true
		} else {
			_, err := strconv.Atoi(val)
			if err == nil {
				goodIndex = true
			}
		}
		if !goodIndex {
			return false, fmt.Errorf("Invalid argument. boot_index must be \"none\" or an integer, got \"boot_index=%s\"", val)
		}
	}

	stringArgsMap[key] = val
	return true, nil
}

func validateInstanceVolumeSubArgCombo(vols *volumeFlagSlice) error {
	errPrefix := "Invalid volume argument combination:"

	for _, v := range *vols {
		if v.swap && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s swap requires either a uuid or size argument", errPrefix)
		}

		if v.ephemeral && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s ephemeral requires either a uuid or size argument", errPrefix)
		}

		if v.local && v.uuid == "" && v.size == 0 {
			return fmt.Errorf("%s local requires either a uuid or size argument", errPrefix)
		}

		if v.uuid != "" && v.size != 0 {
			return fmt.Errorf("%s only one of uuid or size arguments allowed", errPrefix)
		}
		if v.bootIndex != "" && v.uuid == "" {
			return fmt.Errorf("%s boot_index requires a volume uuid", errPrefix)
		}
	}
	return nil
}

func processInstanceVolumeSubArg(subArg string, stringArgsMap map[string]string, boolArgsMap map[string]bool, intArgsMap map[string]int) error {
	if !isInstanceVolumeIntArg(subArg) &&
		!isInstanceVolumeStringArg(subArg) &&
		!isInstanceVolumeBoolArg(subArg) {

		return fmt.Errorf("Invalid argument \"%s\"", subArg)
	}

	ok := getInstanceVolumeImplicitBoolArgs(subArg, boolArgsMap)
	if ok {
		return nil
	}

	// split on "=", put in appropriate map
	keyValue := strings.Split(subArg, "=")
	if len(keyValue) != 2 {
		return fmt.Errorf("Invalid argument. Expected key=value, got \"%s\"", keyValue)
	}
	key := keyValue[0]
	val := keyValue[1]

	ok, err := getInstanceVolumeExplicitBoolArgs(key, val, boolArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	ok, err = getInstanceVolumeIntegerArgs(key, val, intArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	ok, err = getInstanceVolumeStringArgs(key, val, stringArgsMap)
	if err != nil {
		return err
	} else if ok {
		return nil
	}

	return nil
}

var instanceAddCmd = &cobra.Command{
	Use:  "instance [UUID]",
	Long: `Add and launch a specific instance.

The volume flag allows specification of a volume to be attached
to a workload instance.  Sub-options include 'uuid', 'boot_index',
'swap', 'ephemeral', 'local', and 'size'.  Ephemeral volumes are
automatically removed when an instance is removed.  Local volumes
are constrained by a size which is a resource demand considered
when scheduling a workload instance.  Size is an integer number of
gigabytes.  The boot_index may be \"none\" or a negative integer to
exclude the volume from boot, otherwise use a positive integer
to indicate a relative ordering among multiple specified volumes.

Valid combinations include:
	-volume uuid=${UUID}[,boot_index=N]
	-volume uuid=${UUID},swap
	-volume uuid=${UUID},ephemeral[,boot_index=N]
	-volume size=${SIZE}
	-volume ephemeral,size=${SIZE}
	-volume local,ephemeral,size=${SIZE}
	-volume swap,size=${SIZE}
	-volume local,swap,size=${SIZE}

Multiple -volume arguments may be specified per workload instance.

`,
	Run: func(cmd *cobra.Command, args []string) {
		validateAddCommandArgs(instanceFlags)

		var server api.CreateServerRequest
		var servers api.Servers

		populateCreateServerRequest(instanceFlags, &server)

		err := validateCreateServerRequest(server)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating create server request: %s\n", err)
		}

		servers, err = C.CreateInstances(server)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating instances: %s\n", err)
		}

		if C.Template != "" {
			tfortools.OutputToTemplate(os.Stdout, "instance-add", C.Template,
				&servers.Servers, nil)
		}

		if len(servers.Servers) < instanceFlags.instances {
			fmt.Println("Some instances failed to start - check the event log for details.")
		}

		for _, server := range servers.Servers {
			fmt.Printf("Created new (pending) instance: %s\n", server.ID)
		}
	},
}

var poolAddCmd = &cobra.Command{
	Use:  "pool [NAME]",
	Long: `Add a pool to the cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		poolFlags.name = args[0]
		if poolFlags.name == "" {
			fmt.Println("Missing required pool NAME parameter")
			return
		}

		/* Handle the create option so we can add new pools */
		if poolFlags.new {
			err := C.CreateExternalIPPool(poolFlags.name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating pool: %s\n", err)
			}

			fmt.Printf("Created new pool: %s\n", poolFlags.name)
			return
		}

		if poolFlags.subnet != "" {
			// verify it's a good address.
			_, network, err := net.ParseCIDR(poolFlags.subnet)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error validating address: %s\n", err.Error())
			}

			if ones, bits := network.Mask.Size(); bits-ones < 2 {
				fmt.Println("Use address mode to add a single IP address")
			}

			err = C.AddExternalIPSubnet(poolFlags.name, network)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding external IP subnet: %s\n", err)
				return
			}
		} else if len(args) < 1 {
			fmt.Println("Missing any addresses to add")
			return
		} else {
			for _, addr := range args {
				// verify it's a good address
				IP := net.ParseIP(addr)
				if IP == nil {
					fmt.Println("Invalid IP address")
					return
				}
			}
			err := C.AddExternalIPAddresses(poolFlags.name, args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error adding external IP addresses", err)
				return
			}
		}

		fmt.Printf("Added new address to: %s\n", poolFlags.name)
	},
}

var tenantAddCmd = &cobra.Command{
	Use: "tenant [NAME]",
	Long: `Create a new tenant with the supplied flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if !C.IsPrivileged() {
			fmt.Fprintf(os.Stderr, "Creating tenants is only available for privileged users")
		}

		if tenantFlags.tenantID == "" {
			fmt.Fprintf(os.Stderr, "Missing required tenantID")
			return
		}

		// CIDR prefix size must be between 12 and 30 bits
		if tenantFlags.cidrPrefixSize != 0 && (tenantFlags.cidrPrefixSize > 30 || tenantFlags.cidrPrefixSize < 12) {
			fmt.Fprintf(os.Stderr, "cidr-prefix-size must be 12-30")
			return
		}

		var t *template.Template
		if C.Template != "" {
			var err error
			t, err = tfortools.CreateTemplate("tenant-create", C.Template, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				return
			}
		}

		tuuid, err := uuid.Parse(tenantFlags.tenantID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Tenant ID must be a UUID4")
			return
		}

		summary, err := C.CreateTenantConfig(tuuid.String(), tenantFlags.name, tenantFlags.cidrPrefixSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating tenant configuration: %s\n", err)
			return
		}

		if t != nil {
			if err := t.Execute(os.Stdout, &summary); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				return
			}
			return
		}

		fmt.Printf("Tenant [%s]\n", summary.ID)
		fmt.Printf("\tName: %s\n", summary.Name)
	},
}

var volumeAddCmd = &cobra.Command{
	Use:  "volume",
	Long: `Add a volume to a given instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		createReq := api.RequestedVolume{
			Description: volFlags.description,
			Name:        volFlags.name,
			Size:        volFlags.size,
		}

		if volFlags.sourcetype == "image" {
			createReq.ImageRef = volFlags.source
		} else if volFlags.sourcetype == "volume" {
			createReq.SourceVolID = volFlags.source
		} else {
			fmt.Fprintf(os.Stderr, "Unknown source type [%s]\n", volFlags.sourcetype)
		}

		vol, err := C.CreateVolume(createReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating volume: %s\n", err)
		}

		fmt.Fprintf(os.Stderr, "Created new volume: %s\n", vol.ID)
	},
}


func optToReqStorage(opt tool.WorkloadOptions) ([]types.StorageResource, error) {
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
				res.SourceID = disk.Source.ID

				if res.SourceID == "" {
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

func optToReq(opt tool.WorkloadOptions, req *types.Workload) error {
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

	// all default resources are required.
	defaults := opt.Defaults

	r := payloads.RequestedResource{
		Type:  payloads.VCPUs,
		Value: defaults.VCPUs,
	}
	req.Defaults = append(req.Defaults, r)

	r = payloads.RequestedResource{
		Type:  payloads.MemMB,
		Value: defaults.MemMB,
	}
	req.Defaults = append(req.Defaults, r)

	return nil
}

var workloadAddCmd = &cobra.Command{
	Use:  "workload",
	Long: `Add a new workload.`,
	Run: func(cmd *cobra.Command, args []string) {
		var opt tool.WorkloadOptions
		var req types.Workload

		if config == "" {
			fmt.Fprintf(os.Stderr, "Please supply a config for the workload.")
			return
		} 

		f, err := ioutil.ReadFile(config)
		if err != nil {
			fmt.Fprintf(os.Stderr,"Unable to read workload config file: %s\n", err)
			return
		}

		err = yaml.Unmarshal(f, &opt)
		if err != nil {
			fmt.Fprintf(os.Stderr,"Config file invalid: %s\n", err)
			return
		}

		err = optToReq(opt, &req)
		if err != nil {
			fmt.Fprintf(os.Stderr,err.Error())
		}

		workloadID, err := C.CreateWorkload(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating workload %s\n", err)
		}

		fmt.Printf("Created new workload: %s\n", workloadID)
	},
}

var addCmds = []*cobra.Command{imageAddCmd, instanceAddCmd, poolAddCmd, volumeAddCmd, workloadAddCmd}

func init() {
	for _, cmd := range addCmds {
		addCmd.AddCommand(cmd)
	}
	RootCmd.AddCommand(addCmd)

	addCmd.PersistentFlags().StringVarP(&C.Template, "template", "t", "", "Template used to format output")

	imageAddCmd.Flags().StringVar(&imgFlags.file, "file", "", "Image file to upload")
	imageAddCmd.Flags().StringVar(&imgFlags.id, "UUID", "", "Image UUID")
	imageAddCmd.Flags().StringVar(&imgFlags.name, "name", "", "Image name")
	imageAddCmd.Flags().StringVar(&imgFlags.visibility, "visibility", "", "Image visibility (internal,public,private) (default \"private\")")

	instanceAddCmd.Flags().IntVar(&instanceFlags.instances, "instances", 1, "Number of instances to create (default 1)")
	instanceAddCmd.Flags().StringVar(&instanceFlags.label, "label", "", "Set a frame label. This will trigger frame tracing")
	instanceAddCmd.Flags().StringVar(&instanceFlags.name, "name", "", "Name for this instance. When multiple instances are requested this is used as a prefix")
	instanceAddCmd.Flags().Var(&instanceFlags.volumes, "volume", "volume descriptor argument list")
	instanceAddCmd.Flags().StringVar(&instanceFlags.workload, "workload", "", "Workload UUID")

	poolAddCmd.Flags().StringVar(&poolFlags.subnet, "subnet", "", "Subnet in CIDR format")
	poolAddCmd.Flags().BoolVar(&poolFlags.new, "new", false, "Create a new external IP pool")

	volumeAddCmd.Flags().StringVar(&volFlags.description, "description", "", "Volume description")
	volumeAddCmd.Flags().StringVar(&volFlags.name, "name", "", "Volume name")
	volumeAddCmd.Flags().IntVar(&volFlags.size, "size", 1, "Size of the volume in GB (default 1)")
	volumeAddCmd.Flags().StringVar(&volFlags.source, "source", "", "ID of image or volume to clone from")
	volumeAddCmd.Flags().StringVar(&volFlags.sourcetype, "source-type", "", "The type of the source to clone from (default \"image\")")

	workloadAddCmd.Flags().StringVarP(&config, "config", "c", "", "Filename for yaml file describing the workload")
}
