package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type override struct {
	Header string
	Match  string
	Host   string
	Path   string
}

type config struct {
	Name     string
	Path     string
	Host     string
	Override override
}

type upstream struct {
	Proxy http.Handler
	Name  string
}

// Upstream represents the upstream service
type Upstream struct {
	Name     string
	Host     string
	Port     string
	Override override
}

// UpstreamConfig represents all configuration of upstream service
type UpstreamConfig struct {
	Upstreams []Upstream
}

func main() {
	r := mux.NewRouter()

	upstreamConfig, err := loadConfig("/opt/go/src/github.com/go-reverse-proxy/upstream.yaml")
	if err != nil {
		log.Fatal(err)
	}

	mp := map[string]Upstream{}
	for _, u := range upstreamConfig.Upstreams {
		mp[u.Name] = u
	}

	proxies := map[string]http.Handler{}

	r.HandleFunc("/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		prefix := getPrefix(r.Host)
		conf := mp[prefix]

		var (
			ok    bool
			proxy http.Handler
		)

		if proxy, ok = proxies[prefix]; !ok {
			proxy = generateProxy(conf)
			proxies[prefix] = proxy
		}
		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(":80", r))
}

func getPrefix(host string) string {
	tokens := strings.Split(host, ".")
	return tokens[0]
}

// generateProxy generates a reverse proxy from given conf
func generateProxy(u Upstream) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			originHost := fmt.Sprintf("%s:%s", u.Host, u.Port)
			r.Header.Add("X-Forwarded-Host", r.Host)
			r.Header.Add("X-Origin-Host", originHost)
			r.Host = originHost
			r.URL.Host = originHost
			r.URL.Scheme = "http"

			if u.Override.Header != "" && u.Override.Match != "" {
				if r.Header.Get(u.Override.Header) == u.Override.Match {
					r.URL.Path = u.Override.Path
				}
			}
		},
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Millisecond,
			}).Dial,
		},
	}
	return proxy
}

// loadConfig loads the configurations of upstream services from .yaml config file
func loadConfig(config string) (UpstreamConfig, error) {
	upstreamConfig := UpstreamConfig{}

	data, err := ioutil.ReadFile(filepath.Clean(config))
	if err != nil {
		return upstreamConfig, err
	}

	if err := yaml.Unmarshal(data, &upstreamConfig); err != nil {
		return upstreamConfig, err
	}
	return upstreamConfig, nil
}
