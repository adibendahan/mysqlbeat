package beater

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/adibendahan/mysqlbeat/config"

	// mysql go driver
	_ "github.com/go-sql-driver/mysql"
)

// Mysqlbeat  is a struct tol hold the beat config & info
type Mysqlbeat struct {
	beatConfig    *config.Config
	done          chan struct{}
	period        time.Duration
	hostname      string
	port          string
	username      string
	password      string
	password64    []byte
	queries       []string
	querytypes    []string
	deltawildcard string

	oldvalues    common.MapStr
	oldvaluesage common.MapStr
}

// New Creates beater
func New() *Mysqlbeat {
	return &Mysqlbeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

// Config is a function to read config file
func (bt *Mysqlbeat) Config(b *beat.Beat) error {

	// Load beater beatConfig
	err := cfgfile.Read(&bt.beatConfig, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	return nil
}

// base64Decode returns text decoded with base64
func base64Decode(src []byte) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(src))
}

// roundF2I is a function that returns a rounded int64 from a float64
func roundF2I(val float64, roundOn float64) (newVal int64) {
	var round float64

	digit := val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}

	return int64(round)
}

// Setup is a function to setup all beat config & info into the beat struct
func (bt *Mysqlbeat) Setup(b *beat.Beat) error {

	var err error

	if len(bt.beatConfig.Mysqlbeat.Queries) > 0 {

		oldvalues := common.MapStr{"mysqlbeat": "init"}
		oldvaluesage := common.MapStr{"mysqlbeat": "init"}

		bt.oldvalues = oldvalues
		bt.oldvaluesage = oldvaluesage

		// Setting default period if not set
		if bt.beatConfig.Mysqlbeat.Period == "" {
			bt.beatConfig.Mysqlbeat.Period = "10s"
		}

		if bt.beatConfig.Mysqlbeat.DeltaWildCard == "" {
			bt.beatConfig.Mysqlbeat.DeltaWildCard = "__DELTA"
		}

		if len(bt.beatConfig.Mysqlbeat.Queries) != len(bt.beatConfig.Mysqlbeat.QueryTypes) {
			err := fmt.Errorf("error on config file, queries array length != querytypes array length (each query should have a corresponding type on the same index)")
			return err
		}

		bt.queries = bt.beatConfig.Mysqlbeat.Queries
		bt.querytypes = bt.beatConfig.Mysqlbeat.QueryTypes

		logp.Info("Total # of queries to execute: %d", len(bt.queries))

		for index, queryStr := range bt.queries {
			logp.Info("Query #%d (type: %s): %s", index+1, bt.querytypes[index], queryStr)
		}

		bt.period, err = time.ParseDuration(bt.beatConfig.Mysqlbeat.Period)
		if err != nil {
			return err
		}

		if bt.beatConfig.Mysqlbeat.Hostname == "" {
			logp.Info("Hostname not selected, proceeding with '127.0.0.1' as default")
			bt.beatConfig.Mysqlbeat.Hostname = "127.0.0.1"
		}

		if bt.beatConfig.Mysqlbeat.Port == "" {
			logp.Info("Port not selected, proceeding with '3306' as default")
			bt.beatConfig.Mysqlbeat.Port = "3306"
		}

		if bt.beatConfig.Mysqlbeat.Username == "" {
			logp.Info("Username not selected, proceeding with 'mysqlbeat_user' as default")
			bt.beatConfig.Mysqlbeat.Username = "mysqlbeat_user"
		}

		pwdbyte, err := base64Decode(bt.beatConfig.Mysqlbeat.Password64)

		if err != nil {
			return err
		}

		if string(pwdbyte) != "" {
			bt.password = string(pwdbyte)
		} else {
			bt.password = "mysqlbeat_pass"
			logp.Info("Password not selected, proceeding with 'mysqlbeat_pass' as default")
		}

		bt.hostname = bt.beatConfig.Mysqlbeat.Hostname
		bt.port = bt.beatConfig.Mysqlbeat.Port
		bt.username = bt.beatConfig.Mysqlbeat.Username
		bt.deltawildcard = bt.beatConfig.Mysqlbeat.DeltaWildCard

	} else {

		err := fmt.Errorf("there are no queries to execute")
		return err
	}
	return nil
}

// Run is a functions that runs the beat
func (bt *Mysqlbeat) Run(b *beat.Beat) error {
	logp.Info("mysqlbeat is running! Hit CTRL-C to stop it.")

	ticker := time.NewTicker(bt.period)
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		err := bt.beat(b)

		if err != nil {
			return err
		}
	}
}

// readData is a function that connects to the mysql, runs the query and returns the data
func (bt *Mysqlbeat) beat(b *beat.Beat) error {

	connString := bt.username + ":" + bt.password + "@tcp(" + bt.hostname + ":" + bt.port + ")/"

	db, err := sql.Open("mysql", connString)

	if err != nil {
		return err
	}

	for index, queryStr := range bt.queries {

		rows, err := db.Query(queryStr)

		if err != nil {
			return err
		}

		columns, err := rows.Columns()
		if err != nil {
			return err
		}

		values := make([]sql.RawBytes, len(columns))
		scanArgs := make([]interface{}, len(values))

		for i := range values {
			scanArgs[i] = &values[i]
		}

		currentRow := 0
		dtNow := time.Now()

		event := common.MapStr{
			"@timestamp": common.Time(dtNow),
			"type":       b.Name,
		}

		for rows.Next() {

			currentRow++

			if bt.querytypes[index] == "single-row" && currentRow == 1 {

				err = rows.Scan(scanArgs...)
				if err != nil {
					return err
				}

				for i, col := range values {

					strColName := string(columns[i])
					strColValue := string(col)
					strColType := "string"

					nColValue, err := strconv.ParseInt(strColValue, 0, 64)

					if err == nil {
						strColType = "int"
					}

					fColValue, err := strconv.ParseFloat(strColValue, 64)

					if err == nil {
						if strColType == "string" {
							strColType = "float"
						}
					}

					if strings.HasSuffix(strColName, bt.deltawildcard) {

						var exists bool
						_, exists = bt.oldvalues[strColName]

						if !exists {

							bt.oldvaluesage[strColName] = dtNow

							if strColType == "string" {
								bt.oldvalues[strColName] = strColValue
							} else if strColType == "int" {
								bt.oldvalues[strColName] = nColValue
							} else if strColType == "float" {
								bt.oldvalues[strColName] = fColValue
							}

						} else {

							if dtOld, ok := bt.oldvaluesage[strColName].(time.Time); ok {
								delta := dtNow.Sub(dtOld)

								if strColType == "int" {
									var calcVal int64

									oldVal, _ := bt.oldvalues[strColName].(int64)

									if nColValue > oldVal {
										var devRes float64
										devRes = float64((nColValue - oldVal)) / float64(delta.Seconds())
										calcVal = roundF2I(devRes, .5)
									} else {
										calcVal = 0
									}

									event[strColName] = calcVal

									bt.oldvalues[strColName] = nColValue
									bt.oldvaluesage[strColName] = dtNow

								} else if strColType == "float" {
									var calcVal float64

									oldVal, _ := bt.oldvalues[strColName].(float64)

									if fColValue > oldVal {
										calcVal = (fColValue - oldVal) / float64(delta.Seconds())
									} else {
										calcVal = 0
									}

									event[strColName] = calcVal

									bt.oldvalues[strColName] = fColValue
									bt.oldvaluesage[strColName] = dtNow
								} else {
									event[strColName] = strColValue
								}
							}
						}
					} else {
						if strColType == "string" {
							event[strColName] = strColValue
						} else if strColType == "int" {
							event[strColName] = nColValue
						} else if strColType == "float" {
							event[strColName] = fColValue
						}
					}

				}

				rows.Close()

			} else if bt.querytypes[index] == "two-columns" {

				err = rows.Scan(scanArgs...)

				if err != nil {
					return err
				}

				strColName := string(values[0])
				strColValue := string(values[1])
				strColType := "string"

				nColValue, err := strconv.ParseInt(strColValue, 0, 64)

				if err == nil {
					strColType = "int"
				}

				fColValue, err := strconv.ParseFloat(strColValue, 64)

				if err == nil {
					if strColType == "string" {
						strColType = "float"
					}
				}

				if strings.HasSuffix(strColName, bt.deltawildcard) {

					var exists bool
					_, exists = bt.oldvalues[strColName]

					if !exists {

						bt.oldvaluesage[strColName] = dtNow

						if strColType == "string" {
							bt.oldvalues[strColName] = strColValue
						} else if strColType == "int" {
							bt.oldvalues[strColName] = nColValue
						} else if strColType == "float" {
							bt.oldvalues[strColName] = fColValue
						}

					} else {

						if dtOld, ok := bt.oldvaluesage[strColName].(time.Time); ok {
							delta := dtNow.Sub(dtOld)

							if strColType == "int" {
								var calcVal int64

								oldVal, _ := bt.oldvalues[strColName].(int64)

								if nColValue > oldVal {
									var devRes float64
									devRes = float64((nColValue - oldVal)) / float64(delta.Seconds())
									calcVal = roundF2I(devRes, .5)

								} else {
									calcVal = 0
								}

								event[strColName] = calcVal

								bt.oldvalues[strColName] = nColValue
								bt.oldvaluesage[strColName] = dtNow

								//logp.Info("DEBUG: o: %d n: %d time diff: %d calc: %d", oldVal, nColValue, int64(delta.Seconds()), calcVal)

							} else if strColType == "float" {
								var calcVal float64

								oldVal, _ := bt.oldvalues[strColName].(float64)

								if fColValue > oldVal {
									calcVal = (fColValue - oldVal) / float64(delta.Seconds())
								} else {
									calcVal = 0
								}

								event[strColName] = calcVal

								bt.oldvalues[strColName] = fColValue
								bt.oldvaluesage[strColName] = dtNow

							} else {
								event[strColName] = strColValue
							}
						}
					}
				} else {
					if strColType == "string" {
						event[strColName] = strColValue
					} else if strColType == "int" {
						event[strColName] = nColValue
					} else if strColType == "float" {
						event[strColName] = fColValue
					}
				}

			} else if bt.querytypes[index] == "multiple-rows" {
				mevent := common.MapStr{
					"@timestamp": common.Time(time.Now()),
					"type":       b.Name,
				}

				err = rows.Scan(scanArgs...)

				if err != nil {
					return err
				}

				for i, col := range values {

					strColValue := string(col)
					n, err := strconv.ParseInt(strColValue, 0, 64)

					if err == nil {
						mevent[columns[i]] = n
					} else {
						f, err := strconv.ParseFloat(strColValue, 64)

						if err == nil {
							mevent[columns[i]] = f
						} else {
							mevent[columns[i]] = strColValue
						}
					}

				}

				b.Events.PublishEvent(mevent)
				logp.Info("Event sent")
			}
		}

		if bt.querytypes[index] != "multiple-rows" && len(event) > 2 {
			b.Events.PublishEvent(event)
			logp.Info("Event sent")
		}

		if err = rows.Err(); err != nil {
			return err
		}
	}
	defer db.Close()

	return nil
}

// Cleanup is a function that does nothing on this beat :)
func (bt *Mysqlbeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop is a function that runs once the beat is stopped
func (bt *Mysqlbeat) Stop() {
	close(bt.done)
}
