#!/bin/bash
set -x

MYSQL_ROOT_PASSWORD=$MYSQL_ROOT_PASSWORD

# Bootstrap mariadb if it hasn't been started
if [[ ! -d /var/lib/mysql/mysql ]]; then
    /usr/bin/mysql-systemd-start pre
    mysqld_safe &
    mysqladmin --silent --wait=30 ping || exit 1

    # Set root user password
    mysql -e "GRANT ALL PRIVILEGES ON *.* TO \"root\"@\"%\" IDENTIFIED by \"$MYSQL_ROOT_PASSWORD\" WITH GRANT OPTION;"

    # Remove anonymous user access
    mysql -e "DELETE FROM mysql.user WHERE User=\"\";"

    # Remove test database
    mysql -e "DROP DATABASE test;"

    mysqladmin shutdown
fi

mysqld_safe &

tail -f /var/lib/mysql/*err
