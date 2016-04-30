# mysqlbeat
Fully customizable Beat for MySQL server - now with default configuration to build performance dashboard.
This beat can also ship the results of any query defined on the config file to elastic.

## Current status
 This project is in beta stage. In fact this is the first time ever I wrote go code.

## Features

* Connect to any MySQL server and run queries
 * 'single-row' queries will be translated as columnname:value.
 * 'two-columns' will be translated as value-column1:value-column2 for each row.
 * 'multiple-rows' each row will be a document (with columnname:value) - NO DELTA SUPPORT.
 * 'multiple-rows' each row will be a document (with columnname:value) - NO DELTA SUPPORT.
 * 'show-slave-delay' will only send the "Seconds_Behind_Master" column from SHOW SLAVE STATUS
* Any column that ends with the delatwildcard (default is __DELTA) will send delta results, extremely useful for server counters.
  ((newval - oldval)/timediff.Seconds())
 
## How to Build

mysqlbeat uses Glide for dependency management. To install glide see: https://github.com/Masterminds/glide

```shell
$ go build
```

## Default Configuration

Edit mysqlbeat configuration in ```mysqlbeat.yml``` .
You can:
* Add queries to the `queries` array
* Add query types to the `querytypes` array
* Define user/pass to connect to the MySQL - MAKE SURE THE USER ONLY HAS PERMISSIONS TO RUN THE QUERY DESIRED AND NOTHING ELSE.
* Password64 should be saved as a base64 []byte array, to generate the array you can use the following: https://play.golang.org/p/L8Z0lFnzCy (make sure to add a comma between all array numbers)
* Define the column wild card for delta columns

If you choose to use the mysqlbeat as is, just run the following on your MySQL Server:
  `GRANT REPLICATION CLIENT, PROCESS ON *.* TO 'mysqlbeat_user'@'%' IDENTIFIED BY 'mysqlbeat_pass';`

## Template
 The default template is provided, if you add any queries you should update the template accordingly.
 
 To apply the default template run:
 	```curl -XPUT http://<host>:9200/_template/mysqlbeat -d@etc/mysqlbeat-template.json```

## How to use
Just run ```mysqlbeat -c mysqlbeat.yml``` and you are good to go.

## MySQL Performance Dashboard by mysqlbeat
This dashboard created as an addition to the MySQL dashboard provided by packetbeat, use them both.
Run the default configuration provided to get the dashboard below (you should import ```dashboard/mysql_performance_dashboard_by_mysqlbeat.json``` to create the dashboard in Kibana).

![mysql_performance_by_mysqlbeat__dashboard__kibana](https://cloud.githubusercontent.com/assets/2807536/14936629/3a3b88e8-0efa-11e6-87ef-eb864498d3ab.png)


## License
GNU General Public License v2
