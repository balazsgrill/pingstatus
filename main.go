package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-ping/ping"
	"github.com/whiteshtef/clockwork"
)

type target struct {
	Host    string `json:"host"`
	Webhook string `json:"webhook"`
}

type config struct {
	Ping []*target `json:"ping"`
}

type payload struct {
	Trigger string `json:"trigger"`
	Message string `json:"message"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (t *target) send(p *payload) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	resp, err := http.Post(t.Webhook, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}
		resp.Body.Close()
		return fmt.Errorf("webhook returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (t *target) check() (*ping.Statistics, error) {
	p, err := ping.NewPinger(t.Host)
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

func (t *target) ping() {
	s, err := t.check()

	pl := &payload{
		Name:    "ping",
		Trigger: "up",
		Status: "OPERATIONAL",
	}

	if err != nil {
		pl.Trigger = "down"
		pl.Status = "MAJOROUTAGE"
		pl.Message = err.Error()
	} else {
		pl.Message = fmt.Sprintf("Received %d/%d, avg time %d ms", s.PacketsRecv, s.PacketsSent, s.AvgRtt)

		if s.PacketsRecv == 0 {
			pl.Trigger = "down"
			pl.Status = "MAJOROUTAGE"
		} else if s.PacketsRecv < s.PacketsSent {
			pl.Status = "PARTIALOUTAGE"
		}
	}

	err = t.send(pl)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	var configfile string

	flag.StringVar(&configfile, "c", "", "Configuration file")
	flag.Parse()

	c, err := os.ReadFile(configfile)
	if err != nil {
		panic(err)
	}

	var conf config
	err = json.Unmarshal(c, &conf)
	if err != nil {
		panic(err)
	}

	sched := clockwork.NewScheduler()
	sched.Schedule().Every(10).Minutes().Do((&conf).doping)

	(&conf).doping()
	sched.Run()
}

func (c *config) doping() {
	for _, t := range c.Ping {
		go t.ping()
	}
}
