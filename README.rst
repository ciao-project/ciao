CIAO Project README
###################

The :abbr:`Cloud Integrated Advanced Orchestrator (CIAO)` Project's goal is
to provide a secure, scalable, and easy-to-deploy cloud orchestration
system. It is designed to agnostically handle generic workloads of virtual
machines, containers, and bare metal apps. Implemented in the Go language,
it separates logic into ``controller``, ``scheduler``, and ``launcher``
components.  

* The `ciao-controller`_ is responsible for policy choices regarding tenant
  workloads.

* The `ciao-scheduler`_ implements push-based scheduling, finding a "first
  fit" on cluster compute nodes for a controller-approved workload instance.

* The `ciao-launcher`_ abstracts the specific launching details for the different
  workload types (virtual machine, container, bare metal), and reports compute node
  statistics to the scheduler and controller. It also reports per-instance statistics
  up to the controller.

An additional set of componentry provides `networking`_ connectivity for workload
instances and insures tenant isolation.

All CIAO components communicate with each other via `SSNTP`_ using a set of
`payloads`_.

.. _ciao-controller: https://github.com/01org/ciao/blob/master/ciao-controller
.. _ciao-scheduler: https://github.com/01org/ciao/blob/master/ciao-scheduler
.. _ciao-launcher: https://github.com/01org/ciao/blob/master/ciao-launcher
.. _networking: https://github.com/01org/ciao/blob/master/networking
.. _SSNTP: https://github.com/01org/ciao/blob/master/ssntp/README.md
.. _payloads: https://github.com/01org/ciao/blob/master/payloads
