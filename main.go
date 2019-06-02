package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

func main() {
	r := mux.NewRouter()

	mp := map[string]config{
		"ide": config{
			Host: "localhost:9051",
		},
	}

	r.HandleFunc("/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		conf := mp[getPrefix(r.Host)]
		proxy := generateProxy(conf)
		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(":9015", r))
}

func getPrefix(host string) string {
	tokens := strings.Split(host, ".")
	return tokens[0]
}

// generateProxy generates a reverse proxy from given conf
func generateProxy(conf config) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			originHost := conf.Host
			r.Header.Add("X-Forwarded-Host", r.Host)
			r.Header.Add("X-Origin-Host", originHost)
			r.Host = originHost
			r.URL.Host = originHost
			r.URL.Scheme = "http"

			if conf.Override.Header != "" && conf.Override.Match != "" {
				if r.Header.Get(conf.Override.Header) == conf.Override.Match {
					r.URL.Path = conf.Override.Path
				}
			}
		},
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
		},
	}
	return proxy
}
