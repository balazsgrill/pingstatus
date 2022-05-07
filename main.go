package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-ping/ping"
)

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
		stats, err := (&target{Host: host}).check()
		if err != nil {
			log.Printf("Ping error %v\n", err)
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if stats.PacketsRecv < stats.PacketsSent {
			answer := fmt.Sprintf("Received %d packets out of %d", stats.PacketsRecv, stats.PacketsSent)
			log.Println(answer)
			w.WriteHeader(503)
			w.Write([]byte(answer))
			return
		}
		w.Write([]byte("OK"))
	})

	// listen to port
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
