package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/ilyakaznacheev/cleanenv"
	netatmo "github.com/joshuabeny1999/netatmo-api-go"
)

type Config struct {
	Netatmo struct {
		ClientId     string `yaml:"client_id" env:"NETATMO_INFLUX_IMPORTER_NETATMO_CLIENT_ID"`
		ClientSecret string `yaml:"client_secret" env:"NETATMO_INFLUX_IMPORTER_NETATMO_CLIENT_SECRET"`
		Username     string `yaml:"username" env:"NETATMO_INFLUX_IMPORTER_NETATMO_USERNAME"`
		Password     string `yaml:"password" env:"NETATMO_INFLUX_IMPORTER_NETATMO_PASSWORD"`
	} `yaml:"netatmo"`
	Influx struct {
		Url    string `yaml:"url" env:"NETATMO_INFLUX_IMPORTER_INFLUX_URL" env-default:"http://localhost:8086`
		Token  string `yaml:"token" env:"NETATMO_INFLUX_IMPORTER_INFLUX_TOKEN" env-default:"my-token`
		Bucket string `yaml:"bucket" env:"NETATMO_INFLUX_IMPORTER_INFLUX_BUCKET" env-default:"my-bucket`
		Org    string `yaml:"org" env:"NETATMO_INFLUX_IMPORTER_INFLUX_ORG" env-default:"my-org`
	} `yaml:"influx"`
}

func main() {
	if err := run(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	var cfg Config

	configFilename := flag.String("config", "config.yml", "configuration file to parse'")
	flag.Parse()

	if err := cleanenv.ReadConfig(*configFilename, &cfg); err != nil {
		return err
	}
	client := influxdb2.NewClient(cfg.Influx.Url, cfg.Influx.Token)
	writeAPI := client.WriteAPI(cfg.Influx.Org, cfg.Influx.Bucket)
	// always close client at the end
	defer client.Close()

	n, err := netatmo.NewClient(netatmo.Config{
		ClientID:     cfg.Netatmo.ClientId,
		ClientSecret: cfg.Netatmo.ClientSecret,
		Username:     cfg.Netatmo.Username,
		Password:     cfg.Netatmo.Password,
	})
	if err != nil {
		return err
	}

	dc, err := n.Read()
	if err != nil {
		return err
	}

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
