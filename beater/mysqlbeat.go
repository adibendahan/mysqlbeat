package beater

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
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
	passwordAES   string
	queries       []string
	queryTypes    []string
	deltaWildcard string

	oldValues    common.MapStr
	oldValuesAge common.MapStr
}

var (
	commonIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
)

const (
	// secret length must be 16, 24 or 32, corresponding to the AES-128, AES-192 or AES-256 algorithms
	// you should compile your mysqlbeat with a unique secret and hide it (don't leave it in the code after compiled)
	// you can encrypt your password with github.com/adibendahan/mysqlbeat-password-encrypter just update your secret
	// (and commonIV if you choose to change it) and compile.
	secret = "github.com/adibendahan/mysqlbeat"

	defaultPeriod        = "10s"
	defaultHostname      = "127.0.0.1"
	defaultPort          = "3306"
	defaultUsername      = "mysqlbeat_user"
	defaultPassword      = "mysqlbeat_pass"
	defaultDeltaWildcard = "__DELTA"
)

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

// Setup is a function to setup all beat config & info into the beat struct
func (bt *Mysqlbeat) Setup(b *beat.Beat) error {

	if len(bt.beatConfig.Mysqlbeat.Queries) < 1 {
		err := fmt.Errorf("there are no queries to execute")
		return err
	}

	// init the oldValues and oldValuesAge array
	bt.oldValues = common.MapStr{"mysqlbeat": "init"}
	bt.oldValuesAge = common.MapStr{"mysqlbeat": "init"}

	if len(bt.beatConfig.Mysqlbeat.Queries) != len(bt.beatConfig.Mysqlbeat.QueryTypes) {
		err := fmt.Errorf("error on config file, queries array length != queryTypes array length (each query should have a corresponding type on the same index)")
		return err
	}

	// Setting defaults for missing config
	if bt.beatConfig.Mysqlbeat.Period == "" {
		logp.Info("Period not selected, proceeding with '%v' as default", defaultPeriod)
		bt.beatConfig.Mysqlbeat.Period = defaultPeriod
	}

	if bt.beatConfig.Mysqlbeat.Hostname == "" {
		logp.Info("Hostname not selected, proceeding with '%v' as default", defaultHostname)
		bt.beatConfig.Mysqlbeat.Hostname = defaultHostname
	}

	if bt.beatConfig.Mysqlbeat.Port == "" {
		logp.Info("Port not selected, proceeding with '%v' as default", defaultPort)
		bt.beatConfig.Mysqlbeat.Port = defaultPort
	}

	if bt.beatConfig.Mysqlbeat.Username == "" {
		logp.Info("Username not selected, proceeding with '%v' as default", defaultUsername)
		bt.beatConfig.Mysqlbeat.Username = defaultUsername
	}

	if bt.beatConfig.Mysqlbeat.Password == "" && bt.beatConfig.Mysqlbeat.EncryptedPassword == "" {
		logp.Info("Password not selected, proceeding with default password")
		bt.beatConfig.Mysqlbeat.Password = defaultPassword
	}

	if bt.beatConfig.Mysqlbeat.DeltaWildcard == "" {
		logp.Info("DeltaWildcard not selected, proceeding with '%v' as default", defaultDeltaWildcard)
		bt.beatConfig.Mysqlbeat.DeltaWildcard = defaultDeltaWildcard
	}

	// Parse the Period string
	var durationParseError error
	bt.period, durationParseError = time.ParseDuration(bt.beatConfig.Mysqlbeat.Period)
	if durationParseError != nil {
		return durationParseError
	}

	// Handle password decryption and save in the bt
	if bt.beatConfig.Mysqlbeat.Password != "" {
		bt.password = bt.beatConfig.Mysqlbeat.Password
	} else if bt.beatConfig.Mysqlbeat.EncryptedPassword != "" {
		aesCipher, err := aes.NewCipher([]byte(secret))
		if err != nil {
			return err
		}
		cfbDecrypter := cipher.NewCFBDecrypter(aesCipher, commonIV)
		chiperText, err := hex.DecodeString(bt.beatConfig.Mysqlbeat.EncryptedPassword)
		if err != nil {
			return err
		}
		plaintextCopy := make([]byte, len(chiperText))
		cfbDecrypter.XORKeyStream(plaintextCopy, chiperText)
		bt.password = string(plaintextCopy)
	}

	// Save config values to the bt
	bt.hostname = bt.beatConfig.Mysqlbeat.Hostname
	bt.port = bt.beatConfig.Mysqlbeat.Port
	bt.username = bt.beatConfig.Mysqlbeat.Username
	bt.queries = bt.beatConfig.Mysqlbeat.Queries
	bt.queryTypes = bt.beatConfig.Mysqlbeat.QueryTypes
	bt.deltaWildcard = bt.beatConfig.Mysqlbeat.DeltaWildcard

	logp.Info("Total # of queries to execute: %d", len(bt.queries))
	for index, queryStr := range bt.queries {
		logp.Info("Query #%d (type: %s): %s", index+1, bt.queryTypes[index], queryStr)
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

// Cleanup is a function that does nothing on this beat :)
func (bt *Mysqlbeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop is a function that runs once the beat is stopped
func (bt *Mysqlbeat) Stop() {
	close(bt.done)
}

/// *** mysqlbeat methods ***///

// beat is a function that connects to the mysql, runs the query and returns the data
func (bt *Mysqlbeat) beat(b *beat.Beat) error {

	connString := bt.username + ":" + bt.password + "@tcp(" + bt.hostname + ":" + bt.port + ")/"

	db, err := sql.Open("mysql", connString)

	if err != nil {
		return err
	}
	defer db.Close()

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

			if bt.queryTypes[index] == "single-row" && currentRow == 1 {

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

					if strings.HasSuffix(strColName, bt.deltaWildcard) {

						var exists bool
						_, exists = bt.oldValues[strColName]

						if !exists {

							bt.oldValuesAge[strColName] = dtNow

							if strColType == "string" {
								bt.oldValues[strColName] = strColValue
							} else if strColType == "int" {
								bt.oldValues[strColName] = nColValue
							} else if strColType == "float" {
								bt.oldValues[strColName] = fColValue
							}

						} else {

							if dtOld, ok := bt.oldValuesAge[strColName].(time.Time); ok {
								delta := dtNow.Sub(dtOld)

								if strColType == "int" {
									var calcVal int64

									oldVal, _ := bt.oldValues[strColName].(int64)

									if nColValue > oldVal {
										var devRes float64
										devRes = float64((nColValue - oldVal)) / float64(delta.Seconds())
										calcVal = roundF2I(devRes, .5)
									} else {
										calcVal = 0
									}

									event[strColName] = calcVal

									bt.oldValues[strColName] = nColValue
									bt.oldValuesAge[strColName] = dtNow

								} else if strColType == "float" {
									var calcVal float64

									oldVal, _ := bt.oldValues[strColName].(float64)

									if fColValue > oldVal {
										calcVal = (fColValue - oldVal) / float64(delta.Seconds())
									} else {
										calcVal = 0
									}

									event[strColName] = calcVal

									bt.oldValues[strColName] = fColValue
									bt.oldValuesAge[strColName] = dtNow
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

			} else if bt.queryTypes[index] == "two-columns" {

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

				if strings.HasSuffix(strColName, bt.deltaWildcard) {

					var exists bool
					_, exists = bt.oldValues[strColName]

					if !exists {

						bt.oldValuesAge[strColName] = dtNow

						if strColType == "string" {
							bt.oldValues[strColName] = strColValue
						} else if strColType == "int" {
							bt.oldValues[strColName] = nColValue
						} else if strColType == "float" {
							bt.oldValues[strColName] = fColValue
						}

					} else {

						if dtOld, ok := bt.oldValuesAge[strColName].(time.Time); ok {
							delta := dtNow.Sub(dtOld)

							if strColType == "int" {
								var calcVal int64

								oldVal, _ := bt.oldValues[strColName].(int64)

								if nColValue > oldVal {
									var devRes float64
									devRes = float64((nColValue - oldVal)) / float64(delta.Seconds())
									calcVal = roundF2I(devRes, .5)

								} else {
									calcVal = 0
								}

								event[strColName] = calcVal

								bt.oldValues[strColName] = nColValue
								bt.oldValuesAge[strColName] = dtNow

								//logp.Info("DEBUG: o: %d n: %d time diff: %d calc: %d", oldVal, nColValue, int64(delta.Seconds()), calcVal)

							} else if strColType == "float" {
								var calcVal float64

								oldVal, _ := bt.oldValues[strColName].(float64)

								if fColValue > oldVal {
									calcVal = (fColValue - oldVal) / float64(delta.Seconds())
								} else {
									calcVal = 0
								}

								event[strColName] = calcVal

								bt.oldValues[strColName] = fColValue
								bt.oldValuesAge[strColName] = dtNow

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

			} else if bt.queryTypes[index] == "multiple-rows" {
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
			} else if bt.queryTypes[index] == "show-slave-delay" && currentRow == 1 {

				err = rows.Scan(scanArgs...)
				if err != nil {
					return err
				}

				for i, col := range values {

					if string(columns[i]) == "Seconds_Behind_Master" {

						strColName := string(columns[i])
						strColValue := string(col)

						nColValue, err := strconv.ParseInt(strColValue, 0, 64)

						if err == nil {
							event[strColName] = nColValue
						}
					}
					rows.Close()

				}
			}
		}

		if bt.queryTypes[index] != "multiple-rows" && len(event) > 2 {
			b.Events.PublishEvent(event)
			logp.Info("Event sent")
		}

		if err = rows.Err(); err != nil {
			return err
		}
	}

	return nil
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
