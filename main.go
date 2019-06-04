package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-reverse-proxy/proxy"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const FAILED_TIMEOUT = 15

type override struct {
	Header string
	Match  string
	Host   string
	Path   string
}

// Upstream represents upstream host
type Upstream struct {
	Active      bool
	Host        string
	LastAttempt int32
	LastFailure int32
}

// Service represents the backend service
type Service struct {
	Name      string
	Upstreams []Upstream
	Port      string
	Override  override
	Proxy     http.Handler
}

// UpstreamConfig represents all configuration of upstream service
type UpstreamConfig struct {
	Services []Service
}

func main() {
	logger, err := configureLogger("text")
	if err != nil {
		logrus.Fatal(err)
	}

	upstreamConfig, err := loadConfig("/opt/go/src/github.com/go-reverse-proxy/upstream.yaml")
	if err != nil {
		logrus.Fatal(err)
	}

	rr, err := proxy.NewReverseRouter(
		proxy.UseUpstreamConfig(upstreamConfig),
		proxy.UseLogger(logger),
	)

	if err != nil {
		logrus.Fatalln(err)
	}

	logger.Infof("go-reverse-proxy - running on 80, pid: %d", os.Getpid())
	log.Fatal(http.ListenAndServe(":80", rr))
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
