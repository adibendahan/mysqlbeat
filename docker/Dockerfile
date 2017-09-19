FROM ubuntu:16.04

# Install wget (used to pull down beta release of mysqlbeat)
RUN apt-get update && apt-get install -y wget

#Download mysqlbeat release .deb
RUN wget https://github.com/adibendahan/mysqlbeat/releases/download/1.0.0/mysqlbeat_1.0.0-160512235547_amd64.deb && \
    dpkg -i mysqlbeat_1.0.0-160512235547_amd64.deb && \
    rm -rf mysqlbeat_1.0.0-160512235547_amd64.deb

ENTRYPOINT ["/usr/bin/mysqlbeat", "-e"]
