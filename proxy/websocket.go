package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func (rr *ReverseRouter) websocketHandler(w http.ResponseWriter, r *http.Request) {
	name, _ := parseHost(r.Host)
	service := rr.services[name].(*LadleService)
	backendWsURL := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("localhost:%d", service.Port),
	}

	cookies := []string{}
	for _, cookie := range r.Cookies() {
		cookies = append(cookies, cookie.String())
	}

	header := http.Header{
		"Cookie": cookies,
	}

	// websocket connection to backend
	back, _, err := websocket.DefaultDialer.Dial(backendWsURL.String(), header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer back.Close()

	// websocket connection to frontend
	front, err := rr.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer front.Close()

	f2b := make(chan error)
	b2f := make(chan error)

	// goroutine that transfers messages from backend to frontend
	go rr.transfer(front, back, b2f)

	// goroutine that transfers messages from frontend tp backend
	go rr.transfer(back, front, f2b)

	// If either direction fails, finish current websocket session
	select {
	case <-f2b:
		return
	case <-b2f:
		return
	}
}

func (rr *ReverseRouter) transfer(dst, src *websocket.Conn, ch chan error) {
	for {
		if terr := tunnel(dst, src); terr != nil {
			rr.logger.Info(terr.Error())
			ch <- terr
		}
	}
}

func tunnel(dst, src *websocket.Conn) error {
	mt, r, err := src.NextReader()
	if err != nil {
		return err
	}
	w, err := dst.NextWriter(mt)
	if err != nil {
		return err
	}
	defer w.Close()

	if _, cerr := io.Copy(w, r); cerr != nil {
		return cerr
	}
	return nil
}
