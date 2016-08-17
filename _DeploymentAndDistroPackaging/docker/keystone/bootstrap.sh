#!/bin/bash

MARIADB_HOST=$MARIADB_HOST
IDENTITY_HOST=$IDENTITY_HOST
KEYSTONE_DB_USER=$KEYSTONE_DB_USER
KEYSTONE_DB_PASSWORD=$KEYSTONE_DB_PASSWORD
KEYSTONE_DB_NAME=$KEYSTONE_DB_NAME
KEYSTONE_ADMIN_PASSWORD=$KEYSTONE_ADMIN_PASSWORD

sed -i.bak s/IDENTITY_HOST/$IDENTITY_HOST/g /root/openrc
sed -i.bak s/KEYSTONE_ADMIN_PASSWORD/$KEYSTONE_ADMIN_PASSWORD/g /root/openrc
sed -i.bak s/MARIADB_HOST/$MARIADB_HOST/g /etc/keystone/keystone.conf
sed -i.bak s/KEYSTONE_DB_USER/$KEYSTONE_DB_USER/g /etc/keystone/keystone.conf
sed -i.bak s/KEYSTONE_DB_PASSWORD/$KEYSTONE_DB_PASSWORD/g /etc/keystone/keystone.conf
sed -i.bak s/KEYSTONE_DB_NAME/$KEYSTONE_DB_NAME/g /etc/keystone/keystone.conf
sed -i.bak s/IDENTITY_HOST/$IDENTITY_HOST/g /etc/nginx/keystone.wsgi.conf

# Populate keystone database
keystone-manage db_sync

# Nginx & UWSGI
mkdir -p /run/uwsgi/keystone
sed -i 's/uid.*/ /' /usr/share/uwsgi/keystone/{public,admin}.ini
sed -i 's/gid.*/ /' /usr/share/uwsgi/keystone/{public,admin}.ini
echo "logto=/var/log/uwsgi-keystone-admin.log" >> /usr/share/uwsgi/keystone/admin.ini
echo "logto=/var/log/uwsgi-keystone-public.log" >> /usr/share/uwsgi/keystone/public.ini
/usr/bin/uwsgi --ini /usr/share/uwsgi/keystone/admin.ini -s /run/uwsgi/keystone/admin.sock &
/usr/bin/uwsgi --ini /usr/share/uwsgi/keystone/public.ini -s /run/uwsgi/keystone/public.sock &

mkdir /var/lib/nginx
echo "user  root;" >> /usr/share/nginx/conf/nginx.conf
/usr/bin/nginx

/usr/bin/memcached -u root &

# Bootstrap keystone
keystone-manage bootstrap --bootstrap-username admin \
		--bootstrap-password secret \
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
fi

# Create 'user' role
openstack role show user
if [[ $? == 1 ]]; then
    openstack role create user
fi

tail -f /var/log/*
