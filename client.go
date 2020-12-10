package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

type SensuClient struct {
	Name          string   `json:"name"`
	Address       string   `json:"address"`
	Environment   string   `json:"environment"`
	Job           string   `json:"job"`
	Subscriptions []string `json:"subscriptions"`
	KeepAlives    bool     `json:"keepalives"`
	Type          string   `json:"type"`
}

func NewClientFromAlert(alert PromAlert) *SensuClient {
	if alert.Status != "firing" || alert.Labels.Service != "ping" {
		// build client only from firing ping alerts
		return nil
	}
	addr := alert.Labels.IpAddress
	if addr == "" {
		if ip, err := net.LookupHost(alert.Labels.Host); err != nil {
			log.Printf("WARN: lookup %s: %v", alert.Labels.Host, err)
		} else {
			addr = ip[0]
		}
	}

	return &SensuClient{
		Name:          alert.Labels.Host,
		Address:       addr,
		Environment:   alert.Labels.Environment,
		Job:           alert.Labels.Job,
		Subscriptions: []string{"client:" + alert.Labels.Host},
		KeepAlives:    false,
		Type:          "proxy",
	}
}

func (c SensuClient) Update() error {
	buf, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dbg("sensu client update: %+v", c)
	url := fmt.Sprintf("http://%s:%d/clients", *sensuHost, *sensuApiPort)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return fmt.Errorf("post failed: %s", resp.Status)
	}
	dbg("sensu reply: %s", buf)
	return nil
}
