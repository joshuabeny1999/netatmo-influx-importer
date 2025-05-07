package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	netatmo "github.com/joshuabeny1999/netatmo-api-go/v2"
)

// InfluxConfig holds InfluxDB connection parameters.
type InfluxConfig struct {
	URL    string `toml:"url"`
	Token  string `toml:"token"`
	Bucket string `toml:"bucket"`
	Org    string `toml:"org"`
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(stdout io.Writer) error {
	// Flags for separate TOML config files
	influxCfgPath := flag.String("influx-config", "influx.toml", "InfluxDB configuration file (TOML)")
	netatmoCfgPath := flag.String("netatmo-config", "netatmo.toml", "Netatmo configuration file (TOML)")
	flag.Parse()

	// Load InfluxDB configuration
	var influxCfg InfluxConfig
	if _, err := toml.DecodeFile(*influxCfgPath, &influxCfg); err != nil {
		return fmt.Errorf("failed to decode InfluxDB config %q: %w", *influxCfgPath, err)
	}

	// Initialize InfluxDB client
	influxClient := influxdb2.NewClient(influxCfg.URL, influxCfg.Token)
	writeAPI := influxClient.WriteAPI(influxCfg.Org, influxCfg.Bucket)
	defer influxClient.Close()

	// Load Netatmo configuration and client
	nmCfg, err := netatmo.LoadConfig(*netatmoCfgPath)
	if err != nil {
		return fmt.Errorf("failed to load Netatmo config %q: %w", *netatmoCfgPath, err)
	}
	client, err := netatmo.NewClient(nmCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Netatmo client: %w", err)
	}

	// Fetch Netatmo data
	dc, err := client.Read()
	if err != nil {
		return fmt.Errorf("Netatmo Read error: %w", err)
	}

	// Iterate stations and modules, write to InfluxDB
	for _, station := range dc.Stations() {
		for _, module := range station.Modules() {
			if module.DashboardData.LastMeasure == nil {
				fmt.Printf("\t\tSkipping %s, no measurement data available.\n", module.ModuleName)
				continue
			}
			ts, data := module.Info()
			p := influxdb2.NewPointWithMeasurement("city").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", station.Place.City).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)
			p = influxdb2.NewPointWithMeasurement("country").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", station.Place.Country).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)
			p = influxdb2.NewPointWithMeasurement("timezone").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", station.Place.Timezone).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)
			p = influxdb2.NewPointWithMeasurement("longitude").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", *station.Place.Location.Longitude).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)
			p = influxdb2.NewPointWithMeasurement("latitude").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", *station.Place.Location.Latitude).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)
			p = influxdb2.NewPointWithMeasurement("altitude").AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", *station.Place.Altitude).SetTime(time.Unix(ts, 0))
			writeAPI.WritePoint(p)

			for dataName, value := range data {
				p = influxdb2.NewPointWithMeasurement(dataName).AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", value).SetTime(time.Unix(ts, 0))
				writeAPI.WritePoint(p)
			}

			ts, data = module.Data()
			for dataName, value := range data {
				p = influxdb2.NewPointWithMeasurement(dataName).AddTag("station", station.StationName).AddTag("module", module.ModuleName).AddField("value", value).SetTime(time.Unix(ts, 0))
				writeAPI.WritePoint(p)
			}
		}
		writeAPI.Flush()
	}

	return nil
}
