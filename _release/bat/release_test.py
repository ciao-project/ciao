#
# Copyright (c) 2016 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

""" Basic Acceptance Tests for the ciao project

This module contains a set of tests that should be run on a ciao
cluster prior to submitting a pull request. The tests can be
run on a physical cluster, or on the ciao single VM setup.
The output is a TAP (Test Anything Protocol) format file, report.tap.
These tests utilize the python unittest framework.

Prior to running the tests, the following environment variables
must be set:
    "CIAO_IDENTITY" - the URL and port number of your identity service
    "CIAO_CONTROLLER" - the URL and port number of the ciao compute service
    "CIAO_USERNAME" - a test user with user level access to a test tenant
    "CIAO_PASSWORD" - your test user's password
    "CIAO_ADMIN_USERNAME" - your cluster admin user name
    "CIAO_ADMIN_PASSWORD" - your cluster admin pass word.

There are 2 configurable parameters that may be set:

    command_timeout - the length of time the test will wait for a
                      ciao-cli command to return. (default is 30 seconds)
    cluster_timeout - the length of time to wait till an action has occurred
                      in the cluster (default is 60 seconds).

"""
import subprocess32
import os
import unittest
import tap
import time
import sys
import random
import argparse

cli_timeout = 30
retry_count = 60

def ciao_user_env():
    """Sets the user environment up for ciao-cli with the user role

    Copies the current environment and returns an environment
    that ciao-cli will use for user role operations

    Returns:
        an os env dict

    """
    return os.environ.copy()

def ciao_admin_env():
    """Sets the user environment up for ciao-cli with the admin role

    Copies the current environment and returns an environment
    that ciao-cli will use for admin role operations

    Returns:
        an os env dict

    """
    ciao_env = os.environ.copy()
    ciao_env["CIAO_USERNAME"] = ciao_env["CIAO_ADMIN_USERNAME"]
    ciao_env["CIAO_PASSWORD"] = ciao_env["CIAO_ADMIN_PASSWORD"]
    return ciao_env

# implement a wait loop that waits for all instances to move to "active"
# but timeout after so many tries.
def wait_till_active(uuid):
    """Wait in a loop till an instances status has changed to active

    This function will loop for retry_count number of times checking
    the status of an Instance. If the status is not active, it
    will sleep for one second and try again.

    Returns:
        A boolean indicating whether the instance is active or not

    """
    count = retry_count

    while count > 0:
        instances = get_instances()
        for i in instances:
            if i["uuid"] == uuid:
                if i["status"] == "active":
                    return True
                else:
                    break
        time.sleep(1)
        count -= 1

    return False

def launch_workload(uuid, num):
    """Attempt to start a number of instances of a specified workload type

    This function will call ciao-cli and tell it to create an instance
    of a particular workload.

    Args:
        uuid: The workload UUID to start
        num: The number of instances of this workload to start

    Returns:
        A boolean indicating whether the instances were successfully started
    """
    args = ['ciao-cli', 'instance', 'add', '-workload', uuid, '-instances', num]

    try:
        output = subprocess32.check_output(args, env=ciao_user_env(), timeout=cli_timeout)
        lines = output.splitlines()
        line_iter = iter(lines)

        for line in line_iter:
            uuid = line.split(":")[1].lstrip(' ').rstrip()
            done = wait_till_active(uuid)
            if not done:
                return False

        return True

    except subprocess32.CalledProcessError as err:
        print err.output
        return False

def start_random_workload(num="1"):
    """Attempt to start a number of instances of a random workload

    This function will get all the possible workloads, then randomly
    pick one to start.

    Args:
        num: the number of instances to create. Default is 1

    Returns:
        A boolean indicating whether the instances where successfully started
    """
    workloads = get_all_workloads()
    if not workloads:
        return False

    index = random.randint(0, len(workloads) - 1)

    return launch_workload(workloads[index]["uuid"], num)

def launch_all_workloads(num="1"):
    """Attempt to create an instance for all possible workloads

    This function will get all the possible workloads, then
    attempt to create an instance for each one.

    Args:
        num: the number of instances per workload to create. Default is 1

    Returns:
        A boolean indicating whether the instances where successfully started
    """
    workloads = get_all_workloads()
    if not workloads:
        return False

    for workload in workloads:
        # quit on first failure
        success = launch_workload(workload["uuid"], num)
        if not success:
            return False

    return True

def get_all_workloads():
    """Retrieves the list of workload templates from the ciao cluster

    Returns:
        A list of dictionary representations of the workloads found
    """
    args = ['ciao-cli', 'workload', 'list']

    workloads = []

    try:
        output = subprocess32.check_output(args, env=ciao_user_env(), timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return workloads

    lines = output.splitlines()
    line_iter = iter(lines)

    for line in line_iter:
        if line.startswith("Workload"):
            workload = {
                "name": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "image_uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "cpus": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "mem": next(line_iter).split(":")[1].lstrip(' ').rstrip()
            }

            workloads.append(workload)

    return workloads

def get_all_tenants():
    """Retrieves the list of all tenants from the keystone service

    This function uses ciao-cli to get a list of all possible tenants
    from the keystone service. It is called using the admin context.

    Returns:
        A list of dictionary representations of the tenants found
    """
    args = ['ciao-cli', 'tenant', 'list', '-all']

    tenants = []

    try:
        output = subprocess32.check_output(args, env=ciao_admin_env(), timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return tenants

    lines = output.splitlines()
    line_iter = iter(lines)

    for line in line_iter:
        if line.startswith("Tenant"):
            uuid = next(line_iter).split(" ")[1]
            name = next(line_iter).split(" ")[1]
            tenant = {
                "uuid": uuid,
                "name": name
            }
            tenants.append(tenant)

    return tenants

def check_cluster_status():
    """Confirms that the ciao cluster is fully operational

    This function uses ciao-cli to get the list of all compute/network nodes.
    It confirms that the number of ready nodes is equal to the total number of nodes
    It is called with the admin context.

    Returns:
        A boolean indicating whether the cluster is ready or not.
    """
    args = ['ciao-cli', 'node', 'status']

    try:
        output = subprocess32.check_output(args, env=ciao_admin_env(), timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return False

    lines = output.splitlines()
    total = lines[0].split(" ")[2]
    ready = lines[1].split(" ")[1]

    return total == ready

def get_cnci():
    """Gets a list of all CNCIs on the ciao cluster.

    This function is called with the admin context.

    Returns:
        A list of dictionary representations of a CNCI
    """
    args = ['ciao-cli', 'node', 'list', '-cnci']

    cncis = []

    try:
        output = subprocess32.check_output(args, env=ciao_admin_env(), timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return cncis

    lines = output.splitlines()
    line_iter = iter(lines)

    for line in line_iter:
        if line.startswith("CNCI"):
            cnci = {
                "uuid": next(line_iter).split(":")[1],
                "tenant_uuid": next(line_iter).split(":")[1],
                "ip": next(line_iter).split(":")[1]
            }
            cncis.append(cnci)

    return cncis

def delete_all_instances():
    """Deletes all instances for a particular tenant

    This function uses ciao-cli to try to delete all previously created instances.
    It then confirms that the instances were deleted by looping for retry_count
    waiting for the instance to no longer appear in the tenants instance list.

    Returns:
        A boolean indicating that the instances have all been confirmed deleted.
    """
    args = ['ciao-cli', 'instance', 'delete', '-all']

    try:
        output = subprocess32.check_output(args, env=ciao_user_env(), timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return False

    if output.startswith("os-delete"):
        count = retry_count

        while count > 0:
            if len(get_instances()) == 0:
                return True

            time.sleep(1)
            count -= 1

        return False

    return False

def get_instances():
    """Retrieve all created instances for a tenant

    Returns:
        A list of dictionary representations of an instance
    """
    args = ['ciao-cli', 'instance', 'list', '-detail']

    instances = []

    myenv = ciao_user_env()

    try:
        output = subprocess32.check_output(args, env=myenv, timeout=cli_timeout)

    except subprocess32.CalledProcessError as err:
        print err.output
        return instances

    lines = output.splitlines()
    line_iter = iter(lines)

    for line in line_iter:
        if line.startswith("Instance"):
            instance = {
                "uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "status": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "ip": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "mac": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "cn_uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "image_uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip(),
                "tenant_uuid": next(line_iter).split(":")[1].lstrip(' ').rstrip()
            }

            instances.append(instance)

    return instances

class BATTests(unittest.TestCase):
    """Basic Acceptance Tests for the ciao project"""
    def setUp(self):
        pass

    def tearDown(self):
        delete_all_instances()
        time.sleep(2)

    def test_get_tenants(self):
        """Get all tenants"""
        self.failUnless(get_all_tenants())

    def test_cluster_status(self):
        """Confirm that the cluster is ready"""
        self.failUnless(check_cluster_status())

    def test_get_workloads(self):
        """Get all available workloads"""
        self.failUnless(get_all_workloads())

    def test_start_all_workloads(self):
        """Start one instance of all workloads"""
        self.failUnless(launch_all_workloads())

    def test_get_cncis(self):
        """Start a random workload, then get CNCI information"""
        self.failUnless(start_random_workload())
        self.failUnless(get_cnci())

    def test_get_instances(self):
        """Start a random workload, then make sure it's listed"""
        self.failUnless(start_random_workload())
        time.sleep(5)
        instances = get_instances()
        self.failUnless(len(instances) == 1)

    def test_delete_all_instances(self):
        """Start a random workload, then delete it"""
        self.failUnless(start_random_workload())
        self.failUnless(delete_all_instances())
        self.failUnless(not get_instances())


def main():
    """Start the BAT tests

    Confirm that the user has defined the environment variables we need,
    and check for optional arguments. Start the unittests - output in
    TAP format.

    Returns:
        Error if ENV is not set
    """
    global cli_timeout
    global retry_count

    envvars = [
        "CIAO_IDENTITY",
        "CIAO_CONTROLLER",
        "CIAO_USERNAME",
        "CIAO_PASSWORD",
        "CIAO_ADMIN_USERNAME",
        "CIAO_ADMIN_PASSWORD"
    ]

    for var in envvars:
        if var not in os.environ:
            err = "env var %s not set" % var
            sys.exit(err)

    parser = argparse.ArgumentParser(description="ciao Basic Acceptance Tests")
    parser.add_argument("--command_timeout", action="store", dest="cli_timeout",
                        help="Seconds to wait for a command to complete",
                        default=300)
    parser.add_argument("--cluster_timeout", action="store", dest="retry_count",
                        help="Seconds to wait for cluster to respond",
                        default=60)

    args = parser.parse_args()

    cli_timeout = args.cli_timeout
    retry_count = args.retry_count

    outfile = open("./report.tap", "w")
    unittest.main(testRunner=tap.TAPTestRunner(stream=outfile))

if __name__ == '__main__':
    main()
