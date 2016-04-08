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
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/01org/ciao/networking/libsnnet"
)

func main() {
	var err error
	fmt.Println("Args :", os.Args)

	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("	show <name>")
		fmt.Println("	<create|destroy|enable|disable> <bridge|vnic> <name>")
		fmt.Println("	<attach|detach> <bridge_name> <vnic_name>")
		fmt.Println("	create gretap <name> <localIP> <remoteIP> <key>")
		fmt.Println("	destroy gretap <name>")
		fmt.Println("	create instance <name> <cnciIP> <cnIP>")
		fmt.Println("	create conc <name> <cnciIP> <subnet> <cnIP>")
		fmt.Println("	addcn conc <name> <cnciIP> <subnet> <cnIP>")
		fmt.Println("	destroy conc <name> <cnciIP> <subnet> <cnIP>")
		fmt.Println("	test bridge <name>")
		fmt.Println("	test vnic <name>")
		os.Exit(1)
	}

	arg1 := os.Args[1]
	arg2 := os.Args[2]

	if arg1 == "show" {
		if err = libsnnet.Show(arg2); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(os.Args) < 4 {
		fmt.Println("Invalid args", os.Args)
		os.Exit(1)
	}

	arg3 := os.Args[3]

	switch {
	case arg1 == "test" && arg2 == "bridge":
		bridge, _ := libsnnet.NewBridge(arg3)

		if err = bridge.Create(); err == nil {
			if err = bridge.Enable(); err == nil {
				if err = bridge.Disable(); err == nil {
					err = bridge.Destroy()
				}
			}
		}

	case arg1 == "test" && arg2 == "vnic":
		vnic, _ := libsnnet.NewVnic(arg3)

		if err = vnic.Create(); err == nil {
			if err = vnic.Enable(); err == nil {
				if err = vnic.Disable(); err == nil {
					err = vnic.Destroy()
				}
			}
		}

	case arg1 == "create" && arg2 == "bridge":
		bridge, _ := libsnnet.NewBridge(arg3)

		err = bridge.Create()

	case arg1 == "enable" && arg2 == "bridge":
		bridge, _ := libsnnet.NewBridge(arg3)

		if err = bridge.GetDevice(); err == nil {
			err = bridge.Enable()
		}

	case arg1 == "disable" && arg2 == "bridge":
		bridge, _ := libsnnet.NewBridge(arg3)

		if err = bridge.GetDevice(); err == nil {
			err = bridge.Disable()
		}

	case arg1 == "destroy" && arg2 == "bridge":
		bridge, _ := libsnnet.NewBridge(arg3)

		if err = bridge.GetDevice(); err == nil {
			err = bridge.Destroy()
		}

	case arg1 == "create" && arg2 == "vnic":
		vnic, _ := libsnnet.NewVnic(arg3)

		err = vnic.Create()

	case arg1 == "enable" && arg2 == "vnic":
		vnic, _ := libsnnet.NewVnic(arg3)

		if err = vnic.GetDevice(); err == nil {
			vnic.Enable()
		}

	case arg1 == "disable" && arg2 == "vnic":
		vnic, _ := libsnnet.NewVnic(arg3)

		if err = vnic.GetDevice(); err == nil {
			err = vnic.Disable()
		}

	case arg1 == "destroy" && arg2 == "vnic":
		vnic, _ := libsnnet.NewVnic(arg3)
		if err = vnic.GetDevice(); err == nil {
			err = vnic.Destroy()
		}

	case arg1 == "attach":
		bridge, _ := libsnnet.NewBridge(arg2)
		vnic, _ := libsnnet.NewVnic(arg3)

		if err = bridge.GetDevice(); err == nil {
			if err = vnic.GetDevice(); err == nil {
				err = vnic.Attach(bridge)
			}
		}
	case arg1 == "detach":
		bridge, _ := libsnnet.NewBridge(arg2)
		vnic, _ := libsnnet.NewVnic(arg3)

		if err = bridge.GetDevice(); err == nil {
			if err = vnic.GetDevice(); err == nil {
				vnic.Detach(bridge)
			}
		}

	case arg1 == "create" && arg2 == "gretap":
		var key uint64

		id := arg3
		arg4 := os.Args[4]
		arg5 := os.Args[5]
		arg6 := os.Args[6]
		local := net.ParseIP(arg4)
		remote := net.ParseIP(arg5)

		if local == nil || remote == nil {
			err = fmt.Errorf("Bad args for gretap")
		}

		key, err = strconv.ParseUint(arg6, 10, 32)

		if err == nil {
			var gre *libsnnet.GreTunEP
			gre, err = libsnnet.NewGreTunEP(id, local, remote, uint32(key))
			if err == nil {
				err = gre.Create()
			}
		}

	case arg1 == "destroy" && arg2 == "gretap":
		var gre *libsnnet.GreTunEP
		id := arg3
		if gre, err = libsnnet.NewGreTunEP(id, nil, nil, 0); err == nil {
			if err = gre.GetDevice(); err == nil {
				err = gre.Destroy()
			}
		}

	case arg1 == "create" && arg2 == "conc":
		var subnet *net.IPNet

		id := arg3

		tenantUUID := id
		concUUID := id
		cnUUID := os.Args[6]
		subnetUUID := id
		reserved := 10

		cnciIP := net.ParseIP(os.Args[4])
		if cnciIP == nil {
			fmt.Println("Error invalid CNCI IP")
			goto some_error
		}

		if _, subnet, err = net.ParseCIDR(os.Args[5]); err != nil {
			goto some_error
		}
		subnetKey := binary.LittleEndian.Uint32(subnet.IP)
		fmt.Println("Subnet Key := ", subnetKey)

		cnIP := net.ParseIP(os.Args[6])
		if cnIP == nil {
			fmt.Println("Error invalid CN IP")
			goto some_error
		}

		bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
		bridge, _ := libsnnet.NewBridge(bridgeAlias)

		if err = bridge.GetDevice(); err != nil {
			if err = bridge.Create(); err != nil {
				fmt.Println("Error bridge create", err)
				goto some_error
			}
		}

		if err = bridge.Enable(); err != nil {
			fmt.Println("Error bridge enable", err)
			goto some_error
		}

		d, _ := libsnnet.NewDnsmasq(bridgeAlias, tenantUUID, *subnet, reserved, bridge)

		d.Stop() //Ignore any errors

		if err = d.Start(); err != nil {
			fmt.Println("Error starting dnsmasq", err)
			goto some_error
		}

		greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, cnUUID)
		gre, _ := libsnnet.NewGreTunEP(greAlias, cnciIP, cnIP, subnetKey)

		if err = gre.Create(); err != nil {
			fmt.Println("Error gre create", err)
			goto some_error
		}

		if err = gre.Attach(bridge); err != nil {
			fmt.Println("Error gre attach", err)
			goto some_error
		}

		if err = gre.Enable(); err != nil {
			fmt.Println("Error gre enable", err)
			goto some_error
		}
		fmt.Println("Concentrator setup sucessfully")

	case arg1 == "destroy" && arg2 == "conc":
		id := arg3
		tenantUUID := id
		concUUID := id
		cnUUID := os.Args[6]
		subnetUUID := id

		cnciIP := net.ParseIP(os.Args[4])
		if cnciIP == nil {
			fmt.Println("Invalid CNCI IP")
			goto some_error
		}

		var subnet *net.IPNet
		if _, subnet, err = net.ParseCIDR(os.Args[5]); err != nil {
			goto some_error
		}
		subnetKey := binary.LittleEndian.Uint32(subnet.IP)
		fmt.Println("Subnet Key := ", subnetKey)

		cnIP := net.ParseIP(os.Args[6])
		if cnIP == nil {
			fmt.Println("Error invalid CN IP")
			goto some_error
		}

		bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
		bridge, _ := libsnnet.NewBridge(bridgeAlias)

		greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, cnUUID)
		gre, _ := libsnnet.NewGreTunEP(greAlias, cnciIP, cnIP, subnetKey)

		if err = gre.GetDevice(); err != nil {
			fmt.Println("Error gre getdevice", err)
			goto some_error
		}

		if err = gre.Detach(bridge); err != nil {
			fmt.Println("Error gre detach", err)
			goto some_error
		}

		if err = gre.Destroy(); err != nil {
			fmt.Println("Error gre destroy", err)
			goto some_error
		}

		if err = bridge.GetDevice(); err != nil {
			fmt.Println("Warning bridge does not exist", err)
			err = nil
			//goto some_error
		} else {
			var subnet *net.IPNet
			reserved := 10

			if _, subnet, err = net.ParseCIDR(os.Args[5]); err != nil {
				goto some_error
			}

			d, _ := libsnnet.NewDnsmasq(bridgeAlias, tenantUUID, *subnet, reserved, bridge)

			if err = d.Stop(); err != nil {
				fmt.Println("Error cannot stop dnsmasq", err)
			}
			if err = bridge.Destroy(); err != nil {
				fmt.Println("Error bridge destroy", err)
				goto some_error
			}
		}

		fmt.Println("Concentrator deleted sucessfully")

	case arg1 == "create" && arg2 == "instance":
		id := arg3
		cnciIP := net.ParseIP(os.Args[4])
		cnIP := net.ParseIP(os.Args[5])

		tenantUUID := id
		instanceUUID := id
		concUUID := id
		cnUUID := id
		subnetUUID := id
		subnetKey := uint32(0xF)

		bridgeAlias := fmt.Sprintf("br_%s_%s_%s", tenantUUID, subnetUUID, concUUID)
		greAlias := fmt.Sprintf("gre_%s_%s_%s", tenantUUID, subnetUUID, cnUUID)
		vnicAlias := fmt.Sprintf("vnic_%s_%s_%s", tenantUUID, instanceUUID, concUUID)

		if err != nil {
			goto some_error
		}

		bridge, _ := libsnnet.NewBridge(bridgeAlias)

		if err := bridge.Create(); err != nil {
			goto some_error
		}

		if err := bridge.Enable(); err != nil {
			goto some_error
		}

		gre, _ := libsnnet.NewGreTunEP(greAlias, cnIP, cnciIP, subnetKey)

		if err := gre.Create(); err != nil {
			goto some_error
		}

		if err := gre.Attach(bridge); err != nil {
			goto some_error
		}

		if err := gre.Enable(); err != nil {
			goto some_error
		}

		vnic, _ := libsnnet.NewVnic(vnicAlias)

		if err := vnic.Create(); err != nil {
			goto some_error
		}

		if err := vnic.Attach(bridge); err != nil {
			goto some_error
		}

		if err := vnic.Enable(); err != nil {
			goto some_error
		}
		fmt.Println("Instance sucessfully created with name", vnic.LinkName)

	default:
		fmt.Println("Unknown args", os.Args)
		os.Exit(1)
	}

some_error:

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
