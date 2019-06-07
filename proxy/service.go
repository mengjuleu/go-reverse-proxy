package proxy

import (
	"net/http"
	"strings"
)

// LadleService represents a ladle managed service.
// It implements http.Handler interface which can
// serve incoming HTTP request.
type LadleService struct {
	HostName  string
	Override  override
	Proxy     http.Handler
	Port      int
	Upstreams []Upstream
	Static    []struct {
		URI  string
		Path string
	}
}

// ServeHTTP serves incoming HTTP request. It tries to serve static asset first.
// If there is no static asset, then serve the request with backend server.
func (ls *LadleService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, s := range ls.Static {
		if strings.HasPrefix(r.RequestURI, s.URI) {
			http.StripPrefix(s.URI, http.FileServer(http.Dir(s.Path))).ServeHTTP(w, r)
			return
		}
	}
	ls.Proxy.ServeHTTP(w, r)
}
