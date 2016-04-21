Ciao Scheduler
==============

The ciao scheduler implements an
[SSNTP](https://github.com/01org/ciao/tree/master/ssntp) server to
receive workloads from the ciao controller system(s), to receive status
changes regarding caio compute node (CN) resources and launched workload
instances, and to reply to nodes who've checked in by giving them work.



Overview
--------

In addressing the broader problem of dispatching a workload for a
user, ciao splits the problem.

At the top level [ciao-webui](https://github.com/01org/ciao-webui),
[ciao-cli](https://github.com/01org/ciao/tree/master/ciao-cli) and
[ciao-controller](https://github.com/01org/ciao/tree/master/ciao-controller)
are responsible for interacting with user.  Ciao-controller enforces
policy, checking that the users' actions are allowed.  For allowed
actions, ciao-controller sends SSNTP command frames down to
ciao-scheduler.

At the lowest level
[ciao-launcher](https://github.com/01org/ciao/tree/master/ciao-launcher)
is running on each compute node.  It connects to the ciao-scheduler and
sends node level statistics regularly so that the scheduler always knows
the current resource state of the cluster.  The launchers also send up
statistics for each running workload, but scheduler does not pay attention
to these and merely forwards them up the stack to ciao-controller.

This layered design leaves a very lean, scalable scheduler in the middle.

Ciao-scheduler's primary task is to take a new workload description and
find a fit for it in the cluster.  Performing this task entails a search
across only in-memory, known up-to-date data, and is done **very quickly**.



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


Design Thinking
---------------

### A Fit vs. Best Fit

Ciao-scheduler explicitly does not attempt to find the best fit for
a workload.

We bias towards speed of dispatching and simplicity of implementation
over absolute optimality.

Aiming for optimality puts us on a slippery slope which at the extreme
could mean locking all state in the entire cluster, collecting and
analyzing the locked state, making a decision and then unlocking the
state.  This will have bad performance, both in terms of latency to
start an individual workload and for overall throughput when launching
many workloads.

We also assume that while a cloud administrator surely has cost
constraints, they are unlikely to always run a general compute cloud at
the extreme edge of capacity.  If they are providing a service for users,
their users will expect a reasonable responce time for new work orders
and that in turn implies there is indeed capacity for new work.

Finding the best fit is more important if resources are highly constrained
and you want to make an attempt to give future workloads (whose specific
nature is yet unknown) a better chance of succeeding.  Again, though
attempting to address future unknowns adds complexity to the code,
incurs latencies and hinders scalability.

Today a compute node that has no remaining capacity (module a buffer
amount for the launcher and host OS's stability) will report that
it is full and the scheduler will not dispatch work to that node.
As a last resort, ciao-scheduler will return a "cloud full" status to
ciao-controller if no compute nodes have capacity to do work.

### Data Structures and Scale

In the initial implementation, the scheduling choice
focuses primarily on RAM, disk and CPU availability
(see the "Resource" enumeration type in the [start
payload](https://github.com/01org/ciao/blob/master/payloads/start.go) for
more details) on compute nodes relative to the requested workload start.
This list of tracked resource types is will grow over time to encompass
many more compute node and workload characteristics.  We don't expect
that to significantly impact the time needed to make a scheduling choice.
We have designed throughout ciao to scale.

Our goal is to have scheduling take on the order of microseconds.
While we haven't yet tested on extremely large clusters, conceptually one
should expect that searching an in-memory data structure containing many
thousands of nodes' resource data should not take more than milliseconds.
Even if each node is a structure of a thousand unique resource statistics.
And even if the top structure is only a simple linked list.  Walking a
list of thousands of elements and doing thousands of string compares
for each element of the list is not a deeply computationally complex act.

For typical clouds today and in the foreseeable future, we expect our
implementation will scale.

### Robustness and Update-ability

The nature of the launcher agents checking in with scheduler to update
their node statistics and request work means that the scheduler always
has an up-to-date, in-memory representation of the cluster.  No explicit
persistence of this data is required.  The scheduler can crash and
restart, or be stopped and updated and restarted, and launcher agents
will simply reconnect and keep on continually updating the scheduler of
any changes in their node statistics.

### Fairness

Ciao-scheduler currently implements an extremely trivial algorithm to
prefer not using the most-recently-used compute node.  This is inexpensive
and leads to sufficient spread of new workloads across a cluster.



TODO
----

* add temporary queuing of workload starts if all compute node CPU
  utilization is above some level (eg: node load average divided by node
  online cpu count?)
* add knob for cpu overcommit limiting: we observe notable slowness when
  actively starting qemu vCPU's are 40x overcommited to physical CPU's.
* resource estimation: workloads will page in memory on use.  An 8GB VM
  will not necessarily consume all of that 8GB.  Tracking workloads over
  time can allow us to measure real versus requested resource usage.  A RAM
  overcommit knob would allow the gap of unused resource to be overcommited
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
