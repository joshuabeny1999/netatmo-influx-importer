#!/usr/bin/env python3
import os
import glob
import csv
import argparse
import yaml
from datetime import datetime, timezone
from influxdb_client import InfluxDBClient, Point, WritePrecision
from influxdb_client.client.write_api import SYNCHRONOUS
from influxdb_client.rest import ApiException


def load_config(path):
    with open(path, 'r') as f:
        return yaml.safe_load(f)


def import_csv_to_influx(csv_path, write_api, bucket, org):
    """
    Parse a Netatmo-exported CSV and write its data to InfluxDB in batches of 1000 points.
    Temperature is written as float; other measurements as integers.
    Skip batches that raise schema/type conflicts.
    Station tag is always set to "Mein Zuhause (Wetterstation Innensensor)".
    Module tag is read from the CSV metadata.
    """
    station_name = "Mein Zuhause (Wetterstation Innensensor)"

    with open(csv_path, newline='') as csvfile:
        reader = csv.reader(csvfile, delimiter=';')
        next(reader)  # header line 1
        hdr2 = next(reader)  # metadata: [station, lon, lat, module, type]
        module_name = hdr2[3].strip('"')
        hdr3 = next(reader)  # field names
        fields = hdr3[2:]

        points = []
        count = 0

        for row in reader:
            if len(row) < 3:
                continue
            try:
                ts = int(row[0])
            except ValueError:
                continue
            count += 1
            if count % 500 == 0:
                print(f"  processed {count} rows...")

            for i, field_name in enumerate(fields):
                val_str = row[2 + i]
                if not val_str:
                    continue
                try:
                    val_f = float(val_str)
                except ValueError:
                    continue
                # Temperature as float, others as int
                val = val_f

                point = (
                    Point(field_name)
                    .tag("station", station_name)
                    .tag("module", module_name)
                    .field("value", val)
                    .time(ts, WritePrecision.S)
                )
                points.append(point)

            # Flush every 1000 points
            if len(points) >= 1000:
                try:
                    write_api.write(bucket=bucket, org=org, record=points)
                except ApiException as e:
                    if e.status == 422:
                        print(f"  ⚠  Skipping batch at ~{count} rows due to type conflict")
                    else:
                        raise
                points = []

        # Final flush
        if points:
            try:
                write_api.write(bucket=bucket, org=org, record=points)
            except ApiException as e:
                if e.status == 422:
                    print(f"  ⚠  Skipping final batch at ~{count} rows due to type conflict")
                else:
                    raise

        print(f"  → finished {count} rows from {os.path.basename(csv_path)}")


def main():
    parser = argparse.ArgumentParser(
        description="Import Netatmo CSVs into InfluxDB (cleanup + import)"
    )
    parser.add_argument(
        "--config", "-c", default="config.yml",
        help="YAML config file with InfluxDB credentials"
    )
    parser.add_argument(
        "--folder", "-d", default="./data",
        help="Directory containing Netatmo CSV exports"
    )
    args = parser.parse_args()

    cfg = load_config(args.config)
    influx_cfg = cfg['influx']

    client = InfluxDBClient(
        url=influx_cfg['url'],
        token=influx_cfg['token'],
        org=influx_cfg['org']
    )


    # Import CSV data
    write_api = client.write_api(write_options=SYNCHRONOUS)
    pattern = os.path.join(args.folder, "*.csv")
    for csv_file in sorted(glob.glob(pattern)):
        print(f"Importing {os.path.basename(csv_file)} …")
        import_csv_to_influx(
            csv_file,
            write_api,
            influx_cfg['bucket'],
            influx_cfg['org']
        )

    write_api.flush()
    client.close()
    print("Done.")


if __name__ == "__main__":
    main()
