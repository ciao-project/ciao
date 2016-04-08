Ciao Project

Ciao is the "Cloud Integrated Advanced Orchestrator".  Its goal is
to provide an easy to deploy, secure, scalable cloud orchestration
system which handles virtual machines, containers, and bare metal apps
agnostically generic workloads.  Implemented in the Go language, it
separates logic into "controller", "scheduler" and "launcher" components
which communicate over the "Simple and Secure Node Transfer Protocol
(SSNTP)".

Controller is responsible for policy choices around tenant workloads.

Scheduler implements a push scheduling, finding a first fit on cluster
compute nodes for a controller approved workload instance.

Launcher abstracts the specific launching details for the different
workload types (eg: virtual machine, container, bare metal).  Launcher
reports compute node statistics to the scheduler and controller.  It also
reports per-instance statistics up to controller.

An additional set of componentry provides network connectivity for workload
instances and insures tenant isolation.
