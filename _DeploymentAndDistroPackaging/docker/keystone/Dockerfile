FROM clearlinux/mariadb
MAINTAINER obed.n.munoz@intel.com

ARG swupd_args=""

RUN swupd bundle-add openstack-identity openstack-python-clients  $swupd_args
RUN rm -rf /var/lib/swupd/*

# Keystone
RUN mkdir /etc/keystone
COPY keystone.conf /etc/keystone

RUN mkdir -p /etc/nginx/ssl
COPY keystone.wsgi.conf /etc/nginx/

COPY bootstrap.sh /usr/bin
COPY openrc /root/

WORKDIR /root
CMD '/usr/bin/bootstrap.sh'

EXPOSE 5000
EXPOSE 35357
