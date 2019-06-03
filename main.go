package main

import (
	"fmt"
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
		"api": config{
			Host: "localhost:9001",
		},
		"js": config{
			Host: "localhost:9050",
		},
		"scmweb": config{
			Host: "localhost:9023",
		},
		"mleumonster": config{
			Host: "localhost:9000",
		},
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
		fmt.Println(ok)
		proxy.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(":80", r))
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
