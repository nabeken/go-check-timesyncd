# go-check-timesyncd

[![Go](https://github.com/nabeken/go-check-timesyncd/actions/workflows/go.yml/badge.svg)](https://github.com/nabeken/go-check-timesyncd/actions/workflows/go.yml)

go-check-timesyncd is a nagios-compatible `timesyncd` checker plugin written in Go.

## Checks

The following metrics are checked by the plugin. Please see the code for more details.

- Offset
- Packet Count
- Stratum

## Installation

Please download the binary from [releases](https://github.com/nabeken/go-check-timesyncd/releases).

## Usage

```sh
Usage:
  go-check-timesyncd [OPTIONS]

Application Options:
  -w, --warning=  absolute offset time to result in warning (default: 50ms)
  -c, --critical= absolute offset time to result in critical (default: 100ms)

Help Options:
  -h, --help      Show this help message
```
