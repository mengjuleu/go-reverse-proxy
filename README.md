# go-reverse-proxy
[![Build Status](https://travis-ci.org/mengjuleu/go-reverse-proxy.svg?branch=master)](https://travis-ci.org/mengjuleu/go-reverse-proxy)

go-reverse-proxy is a Gloang-based HTTP proxy.

## Install

```bash
go get -u github.com/mengjuleu/go-reverse-proxy
cd $GOPATH/src/github.com/mengjuleu/go-reverse-proxy
make install
```

## Usage

```bash
NAME:
   go-reverse-proxy - A general purpose proxy service

USAGE:
   go-reverse-proxy [global options] command [command options] [arguments...]

VERSION:
   Go ReverseProxy version 0.1

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --lf value, --log-format value  --log-format=json can only use json or text (default: "text") [$LOG_FORMAT]
   -b value, --bind value          (default: ":80") [$BIND]
   -c value, --config value        (default: "/opt/go/src/github.com/go-reverse-proxy/upstream.yaml") [$CONFIG]
   --read-timeout value            (default: 15) [$READTIMEOUY]
   --write-timeout value           (default: 15) [$WRITETIMEOUY]
   --help, -h                      show help
   --version, -v                   print the version
```

## TODO
- Support websocket
- Support TCP connection
- Support Http/2.0
- Support Load Balancing
