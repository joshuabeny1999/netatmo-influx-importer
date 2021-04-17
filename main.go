package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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

	ct := time.Now().UTC().Unix()
	for _, station := range dc.Stations() {

		fmt.Printf("Station : %s\n", station.StationName)
		fmt.Printf("Station Location Information: %s %s %s %f %f\n", station.Place.City, station.Place.Country, station.Place.Timezone, station.Place.Location.Longitude, station.Place.Location.Latitude)

		

		for _, module := range station.Modules() {
			fmt.Printf("\tModule : %s\n", module.ModuleName)

			{
				if module.DashboardData.LastMeasure == nil {
					fmt.Printf("\t\tSkipping %s, no measurement data available.\n", module.ModuleName)
					continue
				}
				ts, data := module.Info()
				for dataName, value := range data {
					fmt.Printf("\t\t%s : %v (updated %ds ago, %d)\n", dataName, value, ct-ts, ts)
				}
			}

			{
				ts, data := module.Data()
				for dataName, value := range data {
					fmt.Printf("\t\t%s : %v (updated %ds ago, %d)\n", dataName, value, ct-ts, ts)
				}
			}
		}
	}

	return nil
}
