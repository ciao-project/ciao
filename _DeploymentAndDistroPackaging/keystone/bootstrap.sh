#!/bin/bash

IDENTITY_HOST="${IDENTITY_HOST:-localhost}"
KEYSTONE_ADMIN_PASSWORD="${KEYSTONE_ADMIN_PASSWORD:-bb915e9ce0ae4b46e82a069b2ef0f8d7}"

sed -i.bak s/IDENTITY_HOST/$IDENTITY_HOST/g /root/openrc
sed -i.bak s/KEYSTONE_ADMIN_PASSWORD/$KEYSTONE_ADMIN_PASSWORD/g /root/openrc

if [[ ! -d /var/lib/mysql/mysql ]]; then
    cp -r -p /data/* /var/lib/mysql/
fi

# MariaDB
mysqld_safe &
mysqladmin --silent --wait=30 ping || exit 1

# Populate keystone database
keystone-manage db_sync

# Start keystone service
/usr/bin/uwsgi --ini /usr/share/uwsgi/keystone/keystone-uwsgi-admin.ini -s /run/uwsgi/keystone/admin.sock &
/usr/bin/uwsgi --ini /usr/share/uwsgi/keystone/keystone-uwsgi-public.ini -s /run/uwsgi/keystone/public.sock &
/usr/sbin/nginx
/usr/bin/memcached -u root &

# Bootstrap keystone
keystone-manage bootstrap --bootstrap-username admin \
		--bootstrap-password $KEYSTONE_ADMIN_PASSWORD \
		--bootstrap-project-name admin \
		--bootstrap-role-name admin \
		--bootstrap-service-name keystone \
		--bootstrap-admin-url "https://$IDENTITY_HOST:35357/v3" \
		--bootstrap-public-url "https://$IDENTITY_HOST:5000/v3" \
		--bootstrap-internal-url "https://$IDENTITY_HOST:5000/v3"

source /root/openrc
# Create 'service' project if it does not exists
openstack project show service
if [[ $? == 1 ]]; then
    openstack project create --domain default --description "Service Project" service
    openstack role add --project service --user admin admin
fi

# Create 'user' role
openstack role show user
if [[ $? == 1 ]]; then
    openstack role create user
fi

# Run post-keystone.sh script, if it exists
if [ -f "/usr/bin/post-keystone.sh" ]; then
   /usr/bin/post-keystone.sh
fi

tail -f /var/lib/mysql/*.err
tail -f /var/log/nginx*
tail -f /var/log/uwsgi/*
