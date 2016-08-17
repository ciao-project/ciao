FROM clearlinux
MAINTAINER leoswaldo.macias@intel.com

ARG swupd_args

RUN swupd bundle-add database-mariadb $swupd_args
RUN rm -rf /var/lib/swupd/*

RUN mkdir /etc/mariadb
COPY my.cnf /etc/mariadb

RUN mkdir /var/lib/mysql
VOLUME /var/lib/mysql

COPY bootstrap.sh /usr/bin
CMD '/usr/bin/bootstrap.sh'

EXPOSE 3306