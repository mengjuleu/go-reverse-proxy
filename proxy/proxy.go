package proxy

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	// FailedTimeout is the duration to retry broken service
	FailedTimeout = 15
	// LadleProjectsPath is the path to ladle-projects folder
	LadleProjectsPath = "/opt/other/ladle-projects/*.service.yaml"
)

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
	Port      int
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
	services       map[string]http.Handler
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
	rr.services = make(map[string]http.Handler)

	for _, s := range loadLadleService(LadleProjectsPath) {
		s.Upstreams = append(s.Upstreams,
			Upstream{
				Host: fmt.Sprintf("%s:%d", "localhost", s.Port),
			},
			Upstream{
				Host: fmt.Sprintf("%s:%d", "shared1.dev.devbucket.org", s.Port),
			},
			Upstream{
				Host: fmt.Sprintf("%s:%d", "shared2.dev.devbucket.org", s.Port),
			},
		)

		s.Proxy = rr.generateProxy(s)

		// Copy s here because s will be overritten in each iteration
		service := s
		rr.services[s.HostName] = &service
	}

	rr.route()
	return rr, nil
}

func (rr *ReverseRouter) route() {
	rr.Router.HandleFunc("/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		name, _ := parseHost(r.Host)
		if _, ok := rr.services[name]; !ok {
			w.WriteHeader(200)
			return
		}
		rr.services[name].ServeHTTP(w, r)
	})
}

// generateProxy generates a reverse proxy from given service
func (rr *ReverseRouter) generateProxy(s LadleService) http.Handler {
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

// loadLadleService loads service config from ladle-projects.
// It filters out service with no port
func loadLadleService(path string) []LadleService {
	services := []LadleService{}
	files, _ := filepath.Glob(path)
	for _, f := range files {
		ladleService := LadleService{}
		data, _ := ioutil.ReadFile(filepath.Clean(f))
		yaml.Unmarshal(data, &ladleService)
		if ladleService.Port > 0 {
			services = append(services, ladleService)
		}
	}
	return services
}

// getHost returns available host from service's upstreams
func getHost(upstreams *[]Upstream) string {
	for _, u := range *upstreams {
		if u.Active {
			return u.Host
		}

		if int32(time.Now().Unix())-u.LastFailure < FailedTimeout {
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

// parseHost parses the incoming request's host and
// returns service name and site
func parseHost(host string) (string, string) {
	if !strings.Contains(host, "devbucket.org") {
		return "", ""
	}

	tokens := strings.Split(host, ".")

	switch len(tokens) {
	case 5:
		// {name}.{site}.dev.devbucket.org
		return tokens[0], tokens[1]
	case 4:
		if tokens[1] == "dev" {
			// {site}.dev.devbucket.org
			return "bb", tokens[0]
		}
		// {name}.{site}.devbucket.org
		return tokens[0], tokens[1]
	case 3:
		// site.devbucket.org
		return "bb", tokens[0]
	}

	return "", tokens[0]
}
