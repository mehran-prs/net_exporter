# Node exporter

[![CircleCI](https://circleci.com/gh/mehran-prs/net_exporter/tree/master.svg?style=shield)][circleci]

Prometheus exporter for hardware and OS metrics exposed by \*NIX kernels, written
in Go with pluggable metric collectors.

## Installation and Usage

`net_exporter` is a fork of [node_exporter](https://github.com/prometheus/node_exporter) to export net stats that are
not
included in the node_exporter, like exporting tcp connections stats based on ASN.

If you are new to Prometheus and `node_exporter` there is
a [simple step-by-step guide](https://prometheus.io/docs/guides/node-exporter/).

The `net_exporter` listens on HTTP port 9200 by default. See the `--help` output for more options.

### Docker

The `net_exporter` is designed to monitor the host system. It's not recommended
to deploy it as a Docker container because it requires access to the host system.

For situations where Docker deployment is needed, some extra flags must be used to allow
the `net_exporter` access to the host namespaces.

Be aware that any non-root mount points you want to monitor will need to be bind-mounted
into the container.

If you start container for host monitoring, specify `path.rootfs` argument.
This argument must match path in bind-mount of host root. The net_exporter will use
`path.procfs` as prefix to access proc filesystem.

```bash
docker run -d \
  --net="host" \
  --pid="host" \
  -v "/:/host:ro,rslave" \
  -v $(pwd)/collector/fixtures/asn.csv:/etc/asn.csv \
  mehranprs/net_exporter:0.1.0 \
  --path.procfs=/host/proc \
  --collector.netstat.asn_file=/etc/asn.csv
```

## Collectors

There is varying support for collectors on each operating system. The tables
below list all existing collectors and the supported systems.

Collectors are enabled by providing a `--collector.<name>` flag.
Collectors that are enabled by default can be disabled by providing a `--no-collector.<name>` flag.
To enable only some specific collector(s), use `--collector.disable-defaults --collector.<name> ...`.

### Enabled by default

| Name     | Description                               | OS|
|----------|-------------------------------------------|---|
|  netstat | Exposes asn statistics from `/proc/net/tcp`. | Linux|

### Filtering enabled collectors

The `node_exporter` will expose all metrics from enabled collectors by default. This is the recommended way to collect
metrics to avoid errors when comparing metrics of different families.

For advanced use the `node_exporter` can be passed an optional list of collectors to filter metrics. The `collect[]`
parameter may be used multiple times. In Prometheus configuration you can use this syntax under
the [scrape config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#<scrape_config>).

```
  params:
    collect[]:
      - foo
      - bar
```

This can be useful for having different Prometheus servers collect specific metrics from nodes.

## Development building and running

Prerequisites:

* [Go compiler](https://golang.org/dl/)
* RHEL/CentOS: `glibc-static` package.

Building:

    git clone https://github.com/mehran-prs/net_exporter.git
    cd net_exporter
    make build
    ./net_exporter <flags>

To see all available configuration flags:

    ./net_exporter -h

## Running tests

    make test

## TLS endpoint

** EXPERIMENTAL **

The exporter supports TLS via a new web configuration file.

```console
./net_exporter --web.config.file=web-config.yml
```

See the [exporter-toolkit https package](https://github.com/prometheus/exporter-toolkit/blob/v0.1.0/https/README.md) for
more details.