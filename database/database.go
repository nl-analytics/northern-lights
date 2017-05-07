package database

import influx "github.com/influxdata/influxdb/client/v2"
import "log"

// Connection is the client for the Influx DB
var Connection influx.Client

// Connect to an influx database
// Returns a database client
func Connect(username string, password string, addr string) influx.Client {

	if Connection == nil {
		var err error
		Connection, err = influx.NewHTTPClient(influx.HTTPConfig{
			Addr:     addr,
			Username: username,
			Password: password,
		})

		if err != nil {
			log.Fatal(err)
		}
	}

	return Connection
}
