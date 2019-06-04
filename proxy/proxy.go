package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// FAILED_TIMEOUT is the duration to retry broken service
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

// ReverseRouter routes reverse proxy traffic
type ReverseRouter struct {
	*mux.Router
	logger         *logrus.Logger
	services       map[string]Service
	upstreamConfig UpstreamConfig
}

// UseLogger sets router's logger
func UseLogger(logger *logrus.Logger) func(*ReverseRouter) error {
	return func(rr *ReverseRouter) error {
		rr.logger = logger
		return nil
	}
}

// UseUpstreamConfig sets config path
func UseUpstreamConfig(config UpstreamConfig) func(*ReverseRouter) error {
	return func(rr *ReverseRouter) error {
		rr.upstreamConfig = config
		return nil
	}
}

// NewReverseRouter creates a router for reverse proxy
func NewReverseRouter(options ...func(*ReverseRouter) error) (*ReverseRouter, error) {
	rr := &ReverseRouter{}

	for _, f := range options {
		if err := f(rr); err != nil {
			return nil, err
		}
	}

	rr.Router = mux.NewRouter()
	rr.services = make(map[string]Service)

	for _, s := range rr.upstreamConfig.Services {
		s.Upstreams = append(s.Upstreams, Upstream{
			Host:        fmt.Sprintf("%s:%s", "localhost", s.Port),
			LastAttempt: 0,
			LastFailure: 0,
		})
		s.Upstreams = append(s.Upstreams, Upstream{
			Host:        fmt.Sprintf("%s:%s", "shared1.dev.devbucket.org", s.Port),
			LastAttempt: 0,
			LastFailure: 0,
		})
		s.Proxy = rr.generateProxy(s)
		rr.services[s.Name] = s
	}

	rr.route()

	return rr, nil
}

func (rr *ReverseRouter) route() {
	rr.Router.PathPrefix("/m/dev/dist/").Handler(http.StripPrefix("/m/dev/dist/", http.FileServer(http.Dir("/opt/python/bitbucket/bitbucket/local/build/dist/"))))
	rr.Router.PathPrefix("/m/dev/css/").Handler(http.StripPrefix("/m/dev/css/", http.FileServer(http.Dir("/opt/python/bitbucket/bitbucket/local/build/css/"))))
	rr.Router.PathPrefix("/m/dev/messages/").Handler(http.StripPrefix("/m/dev/messages/", http.FileServer(http.Dir("/opt/python/bitbucket/bitbucket/local/build/messages/"))))
	rr.Router.PathPrefix("/m/dev/").Handler(http.StripPrefix("/m/dev/", http.FileServer(http.Dir("/opt/python/bitbucket/bitbucket/media/"))))

	rr.Router.HandleFunc("/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == ":" {
			r.Host = "mleumonster.devbucket.org"
		}

		prefix := getPrefix(r.Host)
		if _, ok := rr.services[prefix]; !ok {
			w.WriteHeader(200)
			return
		}
		rr.services[prefix].Proxy.ServeHTTP(w, r)
	})
}

// generateProxy generates a reverse proxy from given conf
func (rr *ReverseRouter) generateProxy(s Service) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			originHost := getHost(&s.Upstreams)
			r.Header.Add("X-Forwarded-Host", r.Host)
			r.Header.Add("X-Origin-Host", originHost)
			r.URL.Host = originHost
			r.URL.Scheme = "http"

			if s.Override.Header != "" && s.Override.Match != "" {
				if r.Header.Get(s.Override.Header) == s.Override.Match {
					r.URL.Path = s.Override.Path
				}
			}
		},
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
		},
		ModifyResponse: func(resp *http.Response) error {
			rr.logger.WithFields(logrus.Fields{
				"Method":   resp.Request.Method,
				"URI":      resp.Request.RequestURI,
				"Status":   resp.StatusCode,
				"Protocol": resp.Proto,
			}).Infof("%s", time.Now().String())
			return nil
		},
	}
	return proxy
}

func getHost(upstreams *[]Upstream) string {
	for _, u := range *upstreams {
		if u.Active {
			return u.Host
		}

		if int32(time.Now().Unix())-u.LastFailure < FAILED_TIMEOUT {
			continue
		}

		u.LastAttempt = int32(time.Now().Unix())
		conn, err := net.DialTimeout("tcp", u.Host, 100*time.Millisecond)

		if err != nil {
			u.LastFailure = int32(time.Now().Unix())
		} else {
			defer conn.Close() // important!
			u.Active = true
			return u.Host
		}
	}

	return ""
}

func getPrefix(host string) string {
	tokens := strings.Split(host, ".")
	return tokens[0]
}
