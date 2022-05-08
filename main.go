package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-ping/ping"
)

type Checker interface {
	Check() (int, string)
}

func New(host string) Checker {
	if strings.Contains(host, ":") {
		return &httpTarget{url: host}
	}
	return &target{Host: host}
}

type target struct {
	Host string `json:"host"`
}

func (t *target) check() (*ping.Statistics, error) {
	p, err := ping.NewPinger(t.Host)
	p.Interval = time.Millisecond * 100
	p.Timeout = time.Second * 2
	if err != nil {
		return nil, err
	}
	p.Count = 4
	err = p.Run()
	if err != nil {
		return nil, err
	}
	return p.Statistics(), nil
}

func (t *target) Check() (int, string) {
	stats, err := t.check()

	if err != nil {
		log.Printf("Ping error %v\n", err)
		return 500, err.Error()
	}
	if stats.PacketsRecv < stats.PacketsSent {
		answer := fmt.Sprintf("Received %d packets out of %d", stats.PacketsRecv, stats.PacketsSent)
		log.Println(answer)
		return 503, answer
	}
	return http.StatusOK, "OK"
}

type httpTarget struct {
	url string
}

func (t *httpTarget) Check() (int, string) {
	r, err := http.Get(t.url)
	if err != nil {
		log.Printf("HTTP error %v\n", err)
		return 500, err.Error()
	}
	log.Printf("%d\n", r.StatusCode)
	return r.StatusCode, r.Status
}

func main() {
	var key string
	var port int

	flag.StringVar(&key, "k", "", "API Key")
	flag.IntVar(&port, "p", 8080, "HTTP Port")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		host := r.URL.Query().Get("host")
		log.Printf("Request on %s\n", host)
		if key == "" {
			log.Println("Key mismatch")
			w.WriteHeader(401)
			return
		}
		if host == "" {
			log.Println("Host not set")
			w.WriteHeader(404)
			return
		}
		t := New(host)

		status, msg := t.Check()
		w.WriteHeader(status)
		w.Write([]byte(msg))
	})

	// listen to port
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
