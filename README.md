# mysqlbeat
Fully customizable Beat for MySQL server.
This beat ships the results of any query defined on the config file to elastic.


## Current status
 This project is in an alpha stage, In fact this is the first time ever I wrote go code.
 Please excuse any rookie mistakes I might have done, fixes are welcome.

## Features

* Connect to any MySQL server and run queries
 * 'single-row' queries will be translated as columnname:value.
 * 'two-columns' will be translated as value-column1:value-column2 for each row.
 * 'multiple-rows' each row will be a document (with columnname:value) - NO DELTA SUPPORT.

* Any column that ends with the delatwildcard (default is __DELTA) will delta results, extremely useful for server counters.
  ((neval - oldval)/timediff.Seconds())
 
## How to Build

```shell
$ go build
```

## Configuration

You must edit mysqlbeat configuration in ```mysqlbeat.yml``` .

* Add queries to the `queries` array
* Add query types to the `querytypes` array
* Define user/pass to connect to the MySQL - MAKE SURE THE USER ONLY HAS PERMISSIONS TO RUN THE QUERY DESIRED AND NOTHING ELSE.
* Password64 should be saved as a base64 []byte array, to generate the array you can use the following: https://play.golang.org/p/L8Z0lFnzCy (make sure to add a comma between all array numbers)
* Define the column wild card for delta columns


## Template
 Since the template depends on the queries you will decide to run, I can't provide a template.
 I recommend completing the configuration and run the mysqlbeat just enough time to create the index, then get the _mapping and adjust it.
 Delete the index and send the new mapping with:
 	curl -XPUT http://<host>:9200/_template/mysqlbeat -d@etc/mysqlbeat-template.json

## How to use

Just run ```mysqlbeat -c mysqlbeat.yml``` and you are good to go.

## License
GNU General Public License v2