#!/bin/bash
# This scripts informs CI about a working Keystone Service

source /root/openrc

for i in {1..5}; do echo "Test #$i"; openstack user list && break || sleep 5; done

openstack token issue


#ec2 section

function createEC2 () {
	openstack ec2 credentials create --user admin
}

function showNewEC2() {
	ID=$(openstack ec2 credentials list | awk '!/Access/ && !/\+/ {print $2}')
	Exist=$(openstack ec2 credentials show $ID | awk ' /access/ {print $2}')
	[[ "$Exist" ]]
}

function deleteEC2() {
	ID=$(openstack ec2 credentials list | awk '!/Access/ && !/\+/ {print $2}')
	openstack ec2 credentials delete $ID
	 Exist=$(openstack ec2 credentials list | grep $ID)
	[[ ! "$Exist" ]]
}

#endpoint section

function createEndPoint () {
	openstack endpoint create identity public http://fakehost:5500/v3 --region RegionOne
}

function showEndPoint () {
	ID=$(openstack endpoint list | awk '/fakehost/ {print $2}')
	[[ "$ID" ]]
}

function deleteEndPoint () {
	ID=$(openstack endpoint list | awk '/fakehost/ {print $2}')
	openstack endpoint delete $ID
	ID=$(openstack endpoint list | awk '/fakehost/ {print $2}')
	[[ ! "$ID" ]]
}

#Roles section

function createProjectAndUser () {
	 openstack project create --domain default --description "Test Project" TestProject
	 openstack user create TestUser --domain default --password secure --email demo@example.com --project TestProject
}

function createRole () {
	openstack role create TestRole
}

function addUserToRole () {
	openstack role add --user TestUser --project TestProject TestRole
}

function listRole() {
	openstack role list | grep TestRole
}

function showRole() {
	openstack role show TestRole
}

function deleteRole() {
	openstack role delete TestRole
	openstack project delete TestProject
	openstack user delete TestUser
}

#Services

function createService () {
	openstack service create --name TestService --description TestDescription image2
}

function showService () {
	ID=$(openstack service list | awk '/image2/ {print $2}')
	openstack service show $ID
}

function deleteService() {
	openstack service delete TestService
	ID=$(openstack service list | awk '/image2/ {print $2}')
	[[ ! "$ID" ]]
}

#Projects

function createProject () {
	  openstack project create --description TestProjectDescr TestProject
}

function showProject() {
	ID=$(openstack project list | awk '/TestProject/ {print $2}')
	openstack project show $ID
}

function deleteProject() {
	openstack project delete TestProject
}

#Users

function createUser () {
	openstack user create --password testUserpassword --email client@example.com TestUser
}

function showUser (){
	ID=$(openstack user list | awk '/TestUser/ {print $2}')
	openstack user show $ID
}

function deleteUser () {
	openstack user delete TestUser
}

#Lets call the functions start the show
echo "#################### EC2 TESTS #########################"
createEC2
showNewEC2
deleteEC2
echo "#################### ENDPOINTS TESTS #########################"
createEndPoint
showEndPoint
deleteEndPoint
echo "#################### ROLES TESTS #########################"
createProjectAndUser
createRole
addUserToRole
listRole
showRole
deleteRole
echo "#################### SERVICES TESTS #########################"
createService
showService
deleteService
echo "#################### PROJECTS TESTS #########################"
createProject
showProject
deleteProject
echo "#################### USERS TESTS #########################"
createUser
showUser
deleteUser
