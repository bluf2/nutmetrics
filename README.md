# nutmetrics

Prometheus exporter for a [NUT](https://networkupstools.org/) server (UPS). One static Go binary, no dependencies,
`scratch` image, runs as non-root.

- Numeric variables: `network_ups_tools_<name>` (same names as `druggeri/nut_exporter`, so existing dashboards work).
- `ups.status`: `network_ups_tools_ups_status{flag="OL"} 1`, one series per flag.
- Text variables (driver versions, battery type, last self-test): labels of `nut_ups_info`.
- `nutmetrics_up`, `nutmetrics_scrape_duration_seconds`; HTTP 500 when the NUT server cannot be read.

```
nutmetrics --nut.server=192.168.1.10 [--nut.ups=ups] [--web.listen-address=:9199]
```

Endpoints: `/metrics` and `/ups_metrics?ups=NAME` (the druggeri path). Without a UPS name it uses the first UPS the
server lists.

## Image

```
docker run -d --network host bluf2/nutmetrics:2026.11.1 --nut.server=192.168.1.10
```

Also at `ghcr.io/<owner>/nutmetrics`. Versions are CalVer (`YYYY.M.N`); pin by digest in production. `latest` follows the
newest release. Linux amd64 and arm64.

## License

MIT
