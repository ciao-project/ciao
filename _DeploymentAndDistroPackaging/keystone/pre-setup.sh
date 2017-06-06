#!/bin/bash

IDENTITY_HOST="${IDENTITY_HOST:-localhost}"
KEYSTONE_DB_USER=keystone
KEYSTONE_DB_PASSWORD=914de29bc82616d7c159eaf9b1f39402
KEYSTONE_DB_NAME=keystone

sed -i.bak s/KEYSTONE_DB_USER/$KEYSTONE_DB_USER/g /etc/keystone/keystone.conf
sed -i.bak s/KEYSTONE_DB_PASSWORD/$KEYSTONE_DB_PASSWORD/g /etc/keystone/keystone.conf
sed -i.bak s/KEYSTONE_DB_NAME/$KEYSTONE_DB_NAME/g /etc/keystone/keystone.conf
sed -i.bak s/IDENTITY_HOST/$IDENTITY_HOST/g /etc/nginx/conf.d/keystone.wsgi.conf

# MariaDB
mkdir /data
echo "datadir = /data" >> /etc/mysql/my.cnf

mysql_install_db --user=mysql --datadir=/data
mysqld_safe &
mysqladmin --silent --wait=30 ping || exit 1

# Keystone Database and user
mysql -e "create database $KEYSTONE_DB_NAME;"
mysql -e "grant all on $KEYSTONE_DB_NAME.* to '$KEYSTONE_DB_USER'@'%' IDENTIFIED by '$KEYSTONE_DB_PASSWORD';"
mysql -e "grant all on $KEYSTONE_DB_NAME.* to '$KEYSTONE_DB_USER'@'localhost' IDENTIFIED BY '$KEYSTONE_DB_PASSWORD';"
mysql -e "FLUSH PRIVILEGES;"

# Populate keystone database
keystone-manage db_sync
keystone-manage fernet_setup --keystone-user keystone --keystone-group keystone
keystone-manage credential_setup --keystone-user keystone --keystone-group keystone
mysqladmin shutdown

# Nginx & UWSGI
rm /etc/nginx/sites-enabled/default
mkdir /var/log/nginx/keystone
chown www-data:adm /var/log/nginx/keystone/
chown keystone:keystone /var/log/keystone
mkdir /var/www/keystone
chown -R keystone:keystone /var/www/keystone

mkdir -p /run/uwsgi/keystone
sed -i 's/uid.*/ /' /usr/share/uwsgi/keystone/keystone-uwsgi-{public,admin}.ini
sed -i 's/gid.*/ /' /usr/share/uwsgi/keystone/keystone-uwsgi-{public,admin}.ini
echo "logto=/var/log/uwsgi/uwsgi-keystone-admin.log" >> /usr/share/uwsgi/keystone/keystone-uwsgi-admin.ini
echo "logto=/var/log/uwsgi/uwsgi-keystone-public.log" >> /usr/share/uwsgi/keystone/keystone-uwsgi-public.ini

#mkdir /var/lib/nginx
sed -i 's/user.*/user root;/g' /etc/nginx/nginx.conf

# Reset MariaDB datadir
sed -i.bak "s/datadir =.*//g" /etc/mysql/my.cnf
rm -rf /var/lib/mysql/*
