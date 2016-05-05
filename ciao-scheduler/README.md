Ciao Scheduler
==============

The ciao scheduler implements an
[SSNTP](https://github.com/01org/ciao/tree/master/ssntp) server to
receive workloads from the ciao controller system(s), to receive status
changes regarding ciao compute node (CN) resources and launched workload
instances, and to reply to nodes who've checked in by giving them work.


Running Scheduler
-----------------

Scheduler does not need to run as root, unlike other ciao components.

Certificates are assumed to be in /etc/pki/ciao, or can be
specified on the command line via the "-cert" and "-cacert"
command line options.  Certificates are created with the
[ciao-cert](https://github.com/01org/ciao/tree/master/ssntp/ciao-cert)
tool.

For debugging or informational purposes glog options are useful.
The "-heartbeat" option emits a simple textual status update of connected
controller(s) and compute node(s).

Of course nothing much interesting happens until you connect at least
a ciao-controller and ciao-launchers also.  See the [ciao cluster setup
guide]() for more information.

### Usage

```shell
Usage of ./ciao-scheduler:
  -alsologtostderr
    	log to standard error as well as files
  -cacert string
    	CA certificate (default "/etc/pki/ciao/CAcert-server-localhost.pem")
  -cert string
    	Server certificate (default "/etc/pki/ciao/cert-server-localhost.pem")
  -cpuprofile string
    	Write cpu profile to file
  -heartbeat
    	Emit status heartbeat text
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

### Example

```shell
$GOBIN/ciao-scheduler --cacert=/etc/pki/ciao/CAcert-ciao-ctl.intel.com.pem --cert=/etc/pki/ciao/cert-Scheduler-ciao-ctl.intel.com.pem --heartbeat
```

More Information
----------------

See
(ciao-scheduler godoc)[https://github.com/01org/ciao/blob/master/ciao-scheduler/scheduler.go]
for more information on the design thinking behind the implementation.

TODO
----

* add temporary queuing of workload starts if all compute node CPU
  utilization is above some level (eg: node load average divided by node
  online cpu count?)
* add knob for cpu overcommit limiting: we observe notable slowness when
  actively starting qemu vCPU's are 40x overcommitted to physical CPU's.
* resource estimation: workloads will page in memory on use.  An 8GB VM
  will not necessarily consume all of that 8GB.  Tracking workloads over
  time can allow us to measure real versus requested resource usage.  A RAM
  overcommit knob would allow the gap of unused resource to be overcommitted
  to other workloads.  The risk with RAM overcommit is workloads failing
  when paging fails, or workloads running from swap instead of RAM and the
  horrible performance that comes with it.  All manner of additional host
  OS possibilities exist to run workloads in ways that reduce guest memory
  waste.
* evacuation: some level of workload migration will be desirable.  At a
  minimum the cloud administrator may want to take a compute node offline
  for maintenance.  They could issue an evacuate command, which would result
  in the scheduler needing to choose a new compute node on which to run each
  workload on the node to be evacuated.  We have ideas on how to implement
  this crudely (stop/restart) and more kindly (live migration), but have
  not yet had time to begin implementation.
* add go test automation
* add more resource type tracking, eg:
  * CPU speeds, sizes, capabilities
  * cache quality of service capacity
  * presence of specific PCI id's (eg: workload specific accelerators)
  * compute node location (eg: geography, rack)
  * VM or docker image cache state (ie: try to preclude network pulls at
    workload launch time)
* NUMA:
  * We really really do not want to export compute node topology up to
    the workload creator.  And we really really do not want consumers
    of the cloud trying to write workloads or applications which have
    to have a specific NUMA hardware topology and are coded to that
    in the hopes of somehow getting better performance...in a general
    purpose compute cloud.  This is highly unlikely to actually lead to
    deterministically better performance for the user workload.
  * Nevertheless it might make sense to articulate you want for example
    your network related workload to only run within a compute node
    on the NUMA node that contains a specific hardware capability
    (ie: PCI id, or...the special high performance NIC).  We could
    address this need by running multiple normal launcher agents on
    a compute node:  one per NUMA node (each started in a cgroup to
    insure their process scheduling and ram placement is node local)
    and one more for the overall node.  So a basic two socket system
    would have three launcher agents.  Each would be aware of resources,
    with resource over counting.  Resources consumed by the whole node
    would automatically be observed as consumed (by something...don't
    need to care what) from the NUMA-node-specific launcher agents.
    Similarly NUMA-node-specific resource consumption would automatically
    be observed as consumed (by something...don't need to care what)
    from the whole compute node launcher agent perspective.
  * And it may not matter anyway for this use case.  The kernel could
    automatically (or launcher explicitly) cause page migration and IRQ/CPU
    affinity in response to use.  Plus if the workload is significant, it
    can get more streaming RAM bandwidth by using both nodes of two socket
    system.
  * Collect experimental data and decide a path.
