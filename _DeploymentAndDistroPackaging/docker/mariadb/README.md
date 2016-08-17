MariaDB
=======
This provides a Clear Linux* MariaDB

Build
-----
```
    docker build -t clearlinux/mariadb .
```

Or just pull it from Dockerhub
---------------------------
```
    docker pull clearlinux/mariadb
```

Start MariaDB Container
-----------------------
```
    YOUR_HOST=`hostname -f`
    docker run --name mariadb -p 3306:3306 -e MYSQL_ROOT_PASSWORD=secret -d clearlinux/mariadb
    # Test it
    mysql -uroot -h $YOUR_HOST -psecret -e "show databases;"
```

Environment Variables
---------------------
- ``MYSQL_ROOT_PASSWORD`` specifies MariaDB root password


Extra Build ARGs
----------------
- ``swupd_args`` Specifies [SWUPD](https://clearlinux.org/documentation/swupdate_how_to_run_the_updater.html) flags

Default build args in Docker are on: https://docs.docker.com/engine/reference/builder/#/arg