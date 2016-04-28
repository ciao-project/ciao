# Simple Node Network Library #

## Overview ##

The Simple Node Network Library (libsnnet) implements a simple SDN controller.
The library implements all the networking setup primitives required in Ciao. 

libsnnet currently provides the following capabilities
- Creation of isolated tenant overlay L2 networks
- Auto assignment of IP Addresses
- Support for Virtual Machine (QEMU)and Container workloads
- Ability to perform inbound and outbound NAT to/from the workloads

It tries to rely on interfaces directly exposed by the kernel vs using user
space tools to ensure maximum portability. The implementation maintains state
on leaf node vs relying on centralized state. It also uses local state to 
perform any network re-configuration in the event of a launcher crash or restart

Currently the library supports creation of bridges, GRE tunnels, VM and Container
compatible interfaces (VNICs) on nodes. It also provides and the ability to 
attach tunnels and VNICs to bridges.

The implementation also provides the ability to interconnect these bridges
across nodes creating L2 Overlay networks.


## Roles ##

The library supports node specific networking initialization capabilities.
It currently supports setup of Compute Nodes (CN), Network Nodes (NN) and 
Compute Node Concentrator Instances (CNCI)

### Compute Node ###

A compute node typically runs VM and Container workloads. The library provides
API's to perform network initialization as well as network interface creation
and overlay network linking.

### Network Node ###

The tenant overlay networks are linked together to Network Nodes. The Network
Node switch and route traffic between the tenant bridges and subnets distributed
across multiple Compute Nodes.

### CNCI ###

Compute Node Concentrators or CNCIs are Virtual Machines running on
Network Nodes which handle subsets of traffic belonging to a single tenant.
A single network node can run multiple CNCI's limited by the Compute and 
Network needs of the CNCIs. All tenant level switching and routing for 
a given tenant is handled isolated from other tenants using the CNCI's.
The CNCIs also implement tenant specific firewall and NAT rules. In the future
they may be extended to perform traffic shaping.
