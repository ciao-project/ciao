FROM ubuntu:16.04
MAINTAINER erick.cardona.ruiz@intel.com

# openstack packages
RUN apt-get update
RUN apt-get install -y software-properties-common
RUN add-apt-repository cloud-archive:ocata
RUN apt-get update && apt-get dist-upgrade -y
RUN apt-get install -y python-openstackclient
RUN apt-get install -y mariadb-server python-pymysql
RUN apt-get install -y memcached python-memcache
RUN apt-get install -y keystone nginx libgd-tools nginx-doc python-django-uwsgi uwsgi uwsgi-plugin-python

# MariaDB config
COPY my.cnf /etc/mysql/my.cnf

# Keystone config
RUN mkdir -p /var/tmp
COPY keystone.conf /etc/keystone/

RUN mkdir -p /etc/nginx/ssl
RUN mkdir -p /usr/share/uwsgi/keystone
COPY keystone.wsgi.conf /etc/nginx/conf.d/
COPY keystone-uwsgi-admin.ini /usr/share/uwsgi/keystone
COPY keystone-uwsgi-public.ini /usr/share/uwsgi/keystone
COPY pre-setup.sh /usr/bin
COPY bootstrap.sh /usr/bin
COPY verify-keystone.sh /usr/bin
COPY openrc /root/
RUN /usr/bin/pre-setup.sh

VOLUME /var/lib/mysql
WORKDIR /root
CMD '/usr/bin/bootstrap.sh'

EXPOSE 5000
EXPOSE 35357
