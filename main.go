package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-reverse-proxy/proxy"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/yaml.v2"
)

const (
	defaultPort         = 80
	defaultReadTimeout  = 15
	defaultWriteTimeout = 15
)

func main() {
	var (
		logFormat    string
		bind         string
		configFile   string
		readTimeout  int
		writeTimeout int
	)

	app := cli.NewApp()
	app.Version = "Go ReverseProxy version 0.1"
	app.Usage = "A general purpose proxy service"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "lf,log-format",
			Destination: &logFormat,
			Usage:       "--log-format=json can only use json or text",
			EnvVar:      "LOG_FORMAT",
			Value:       "text",
		},
		cli.StringFlag{
			Name:        "b, bind",
			Destination: &bind,
			EnvVar:      "BIND",
			Value:       fmt.Sprintf(":%d", defaultPort),
		},
		cli.StringFlag{
			Name:        "c, config",
			Destination: &configFile,
			EnvVar:      "CONFIG",
			Value:       "/opt/go/src/github.com/go-reverse-proxy/upstream.yaml",
		},
		cli.IntFlag{
			Name:        "read-timeout",
			Destination: &readTimeout,
			EnvVar:      "READTIMEOUY",
			Value:       defaultReadTimeout,
		},
		cli.IntFlag{
			Name:        "write-timeout",
			Destination: &writeTimeout,
			EnvVar:      "WRITETIMEOUY",
			Value:       defaultWriteTimeout,
		},
	}

	app.Action = func(c *cli.Context) error {
		logger, err := configureLogger(logFormat)
		if err != nil {
			return err
		}

		upstreamConfig, err := loadConfig(configFile)
		if err != nil {
			return err
		}

		rr, err := proxy.NewReverseRouter(
			proxy.UseUpstreamConfig(upstreamConfig),
			proxy.UseLogger(logger),
		)

		if err != nil {
			logrus.Fatalln(err)
		}

		logger.Infof("go-reverse-proxy - running on %s, pid: %d", bind, os.Getpid())

		srv := &http.Server{
			Handler:      rr,
			Addr:         bind,
			WriteTimeout: time.Duration(readTimeout) * time.Second,
			ReadTimeout:  time.Duration(writeTimeout) * time.Second,
		}
		return srv.ListenAndServe()
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

// loadConfig loads the configurations of upstream services from .yaml config file
func loadConfig(config string) (proxy.UpstreamConfig, error) {
	upstreamConfig := proxy.UpstreamConfig{}

	data, err := ioutil.ReadFile(filepath.Clean(config))
	if err != nil {
		return upstreamConfig, err
	}

	if err := yaml.Unmarshal(data, &upstreamConfig); err != nil {
		return upstreamConfig, err
	}
	return upstreamConfig, nil
}

func configureLogger(format string) (*logrus.Logger, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	switch format {
	case "json":
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{logrus.FieldKeyMsg: "message"}}
	case "text":
		logger.Formatter = &logrus.TextFormatter{}
	default:
		return nil, errors.New("Invalid log format value")
	}

	return logger, nil
}
