# Intro

Hi there!

Here is a ready-to-deploy IP geolocation server. It works for both IPv4 and IPv6.
However, the underlying database isn't very large.
It uses the [OpenGeoFeed database](https://github.com/sapics/ip-location-db/blob/master/geo-whois-asn-country/README.md#geofeed-database-update-daily), available [here](https://github.com/sapics/ip-location-db).

# How to Use

Here's a command to run it using Docker:

```bash
docker run -p 8080:8080 ghcr.io/realchandan/ip-geo-api
```

## Request

```bash
curl localhost:8080/getIpInfo?addr=140.82.114.3
```

## Response

```json
{ "ok": true, "country": "US", "ip_addr": "140.82.114.3", "ip_v6": false }
```

# Configuration

You can set `AUTO_UPDATE=true` as an environment variable to make the program check for updates every time.
Also, be sure to mount `/app/data` as a Docker volume so downloaded CSVs can be saved.

# License

The database is licensed under [CC0](https://creativecommons.org/share-your-work/public-domain/cc0/).

This means:

> CC0 doesn't legally require users of the data to cite the source!

But feel free to attribute the IP database [provider](https://opengeofeed.org/)! ❤️

The code in this repository is licensed under the MIT License.

# Thanks!
