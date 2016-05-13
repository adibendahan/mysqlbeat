# mysqlbeat
Fully customizable Beat for MySQL server - this beat can ship the results of any query defined on the config file to Elastic.


## Current status
 First beta release [available here](https://github.com/adibendahan/mysqlbeat/releases/tag/1.0.0).

## Features

* Connect to any MySQL server and run queries
 * `single-row` queries will be translated as columnname:value.
 * `two-columns` will be translated as value-column1:value-column2 for each row.
 * `multiple-rows` each row will be a document (with columnname:value) - no DELTA support.
 * `show-slave-delay` will only send the "Seconds_Behind_Master" column from `SHOW SLAVE STATUS;`
* Any column that ends with the delatwildcard (default is __DELTA) will send delta results, extremely useful for server counters.
  `((newval - oldval)/timediff.Seconds())`
* MySQL Performance Dashboard (more details below)

## How to Build

mysqlbeat uses Glide for dependency management. To install glide see: https://github.com/Masterminds/glide

```shell
$ glide update --no-recursive
$ make
```

## Default Configuration

Edit mysqlbeat configuration in ```mysqlbeat.yml``` .
You can:
 * Add queries to the `queries` array
 * Add query types to the `querytypes` array
 * Define Username/Password to connect to the MySQL
 * Define the column wild card for delta columns
 * Password can be saved in clear text/AES encryption

If you choose to use the mysqlbeat as is, just run the following on your MySQL Server:
  ```
   GRANT REPLICATION CLIENT, PROCESS ON *.* TO 'mysqlbeat_user'@'%' IDENTIFIED BY 'mysqlbeat_pass';
  ```

Notes on password encryption: Before you compile your own mysqlbeat, you should put a new secret in the code (defined as a const), secret length must be 16, 24 or 32, corresponding to the AES-128, AES-192 or AES-256 algorithm. I recommend deleting the secret from the source code after you have your compiled mysqlbeat. You can encrypt your password with [mysqlbeat-password-encrypter](github.com/adibendahan/mysqlbeat-password-encrypter, "github.com/adibendahan/mysqlbeat-password-encrypter") just update your secret (and commonIV if you choose to change it) and compile.

## Template
 The default template is provided, if you add any queries you should update the template accordingly.
 
 To apply the default template run:
 	```
 	 curl -XPUT http://<host>:9200/_template/mysqlbeat -d@etc/mysqlbeat-template.json
 	```

## How to use
Just run ```mysqlbeat -c mysqlbeat.yml``` and you are good to go.

## MySQL Performance Dashboard by mysqlbeat
This dashboard created as an addition to the MySQL dashboard provided by packetbeat, use them both.
Run the default configuration provided to get the dashboard below (you should import ```dashboard/mysql_performance_dashboard_by_mysqlbeat.json``` to create the dashboard in Kibana).

![mysql_performance_by_mysqlbeat__dashboard__kibana](https://cloud.githubusercontent.com/assets/2807536/14936629/3a3b88e8-0efa-11e6-87ef-eb864498d3ab.png)


## License
GNU General Public License v2
