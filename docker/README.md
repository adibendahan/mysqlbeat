# mysqlbeat-docker
Dockerized version of https://github.com/adibendahan/mysqlbeat 

## Build Instructions

```shell
$ git clone https://github.com/turova/mysqlbeat-docker
$ cd mysqlbeat-docker
$ ./build.sh
```

## Run Instructions

If mysqlbeat-docker will be run on the same machine as ElasticSearch or MySql, the Docker gateway IP. On an Ubuntu system with a standard Docker setup, this can typically be retrieved via ```ip addr show | grep "inet.*docker0" | awk '{print $2}' | cut -d '/' -f1```. If either service is configured to run on a different machine, it's normal IP address can be used.

```shell
# Switch to directory with a correctly configured mysqlbeat.yml
$ docker run -it -v $PWD/mysqlbeat.yml:/usr/bin/mysqlbeat.yml mysqlbeat-docker
```

The above line will run ```/usr/bin/mysqlbeat -e``` inside the container using the mysqlbeat.yml file in the local directory

