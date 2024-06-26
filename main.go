package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gopkg.in/yaml.v2"

	netatmo "github.com/joshuabeny1999/netatmo-api-go/v2"
)

type Config struct {
	Netatmo struct {
		ClientID        string    `yaml:"client_id"`
		ClientSecret    string    `yaml:"client_secret"`
		AccessToken     string    `yaml:"access_token"`
		TokenValidUntil time.Time `yaml:"token_valid_until"`
		RefreshToken    string    `yaml:"refresh_token"`
	} `yaml:"netatmo"`
	Influx struct {
		URL    string `yaml:"url"`
		Token  string `yaml:"token"`
		Bucket string `yaml:"bucket"`
		Org    string `yaml:"org"`
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

	data, err := os.ReadFile(*configFilename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}

	client := influxdb2.NewClient(cfg.Influx.URL, cfg.Influx.Token)
	writeAPI := client.WriteAPI(cfg.Influx.Org, cfg.Influx.Bucket)
	// always close client at the end
	defer client.Close()

	n, err := netatmo.NewClient(netatmo.Config{
		ClientID:        cfg.Netatmo.ClientID,
		ClientSecret:    cfg.Netatmo.ClientSecret,
		RefreshToken:    cfg.Netatmo.RefreshToken,
		AccessToken:     cfg.Netatmo.AccessToken,
		TokenValidUntil: cfg.Netatmo.TokenValidUntil,
	})
	if err != nil {
		return err
	}

	if cfg.Netatmo.RefreshToken != n.RefreshToken {
		cfg.Netatmo.RefreshToken = n.RefreshToken

		// Speichern Sie die Konfiguration zurück in die Datei
		data, err = yaml.Marshal(&cfg)
		if err != nil {
			return err
		}
		err = os.WriteFile(*configFilename, data, 0644)
		if err != nil {
			return err
		}
	}

	if cfg.Netatmo.AccessToken != n.AccessToken {
		cfg.Netatmo.AccessToken = n.AccessToken
		cfg.Netatmo.TokenValidUntil = n.TokenValidUntil

		data, err = yaml.Marshal(&cfg)
		if err != nil {
			return err
		}
		err = os.WriteFile(*configFilename, data, 0644)
		if err != nil {
			return err
		}
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
