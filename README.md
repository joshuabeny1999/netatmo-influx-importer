# netatmo-influx-importer

**Go CLI** for cron-driven import of Netatmo weather station data into InfluxDB 2.x.

## Usage

1. **Install**
    - Download the latest release from GitHub Releases:  
      https://github.com/joshuabeny1999/netatmo-influx-importer/releases/latest
    - Extract the binary: `tar -xvzf netatmo-influx-importer_x.y.z_Linux_amd64.tar.gz`
    - Make executable: `chmod +x netatmo-influx-importer`

2. **Configuration**

   You now need two **TOML** files in the same folder as the binary (or point to them via flags):

   ### `netatmo.toml`
   Holds your Netatmo OAuth2 credentials & token state. The CLI will auto-refresh tokens and write back any updates. To get an Token:
   - [Create a new netatmo app](https://dev.netatmo.com/apps/createanapp#form)
   - Generate a new token using the token generator. Scope needed is `read_station`: ![token_generator_netatmo.png](token_generator_netatmo.png)

      ```toml
      client_id         = "YOUR_NETATMO_CLIENT_ID"
      client_secret     = "YOUR_NETATMO_CLIENT_SECRET"
      access_token      = "YOUR_CURRENT_ACCESS_TOKEN"
      refresh_token     = "YOUR_CURRENT_REFRESH_TOKEN"
      token_valid_until = 2025-05-07T15:04:05Z
      ```

      ### `influx.toml`
      InfluxDB connection parameters.
      ```toml
      url    = "http://localhost:8086"
      token  = "YOUR_INFLUXDB_TOKEN"
      bucket = "YOUR_BUCKET"
      org    = "YOUR_ORG"
      ```

3. **Run**

   ```sh
   # default file names netatmo.toml & influx.toml
   ./netatmo-influx-importer

   # specify custom paths
   ./netatmo-influx-importer \
     --netatmo-config /path/to/netatmo.toml \
     --influx-config /path/to/influx.toml
   ```

4. **Schedule via cron** (every 5 minutes recommended)

   ```cron
   */5 * * * * /full/path/to/netatmo-influx-importer
   ```

## Background & History

- **Dec 2023**: Netatmo rolled refresh tokens on every request. Make sure the config file is writable so new tokens persist.
- **Jun 2024**: Added token expiration check to avoid over-refresh.
- **May 2025**: Moved Netatmo config handling into the Go API; switched config formats from YAML to TOML for simpler, unambiguous parsing.

## Troubleshooting

- **Permission errors** writing `netatmo.toml`? Ensure the user running the cron job has write access to that file.
- **Invalid tokens**? Regenerate your Netatmo tokens (via the developer token generator with `read_station` scope) and update `netatmo.toml`.