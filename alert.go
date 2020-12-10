package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type PromAlerts struct {
	Alerts []PromAlert `json:"alerts"`
}

type PromAlert struct {
	Status      string      `json:"status"`
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
	StartsAt    time.Time   `json:"startsAt"`
	EndsAt      time.Time   `json:"endsAt"`
}

type Labels struct {
	AlertName   string `json:"alertname"`
	Environment string `json:"environment"`
	Host        string `json:"instance"`
	IpAddress   string `json:"ip_address"`
	Job         string `json:"job"`
	Service     string `json:"service"`
	Severity    string `json:"severity"`
	Component   string `json:"component"`
}

type Annotations struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type Status int

const (
	OK Status = iota
	WARNING
	CRITICAL
	UNDEF
)

type SensuAlert struct {
	Name   string `json:"name"`
	Output string `json:"output"`
	Source string `json:"source"`
	Status Status `json:"status"`
}

func (p PromAlert) ToSensu() *SensuAlert {
	alert := SensuAlert{
		Name:   p.Labels.AlertName,
		Source: strings.Split(p.Labels.Host, ":")[0],
		Status: toStatus(p.Labels.Severity),
	}
	if p.Labels.Component != "" {
		alert.Name += "_" + strings.NewReplacer("node ", "", "interface ", "", " ", "_", "/", "-", ":", "-").Replace(p.Labels.Component)
		log.Printf("** component=%s -> alert name=%s", p.Labels.Component, alert.Name)
	}
	switch p.Status {
	case "firing":
		alert.Output = fmt.Sprintf("%s: %s", alert.Status, p.Annotations.Value)
	case "resolved":
		if p.Labels.Severity == "ok" {
			// skip ping_ok resolution
			return nil
		}
		if strings.HasPrefix(p.Labels.AlertName, "wdm_laser_") {
			// manual resolution for wdm laser variations alarms
			log.Printf("skip resolution of %s on %s (%s)", alert.Name, p.Labels.Host[:4], p.Annotations.Value)
			return nil
		}
		alert.Status = OK
		alert.Output = "OK: Resolved (" + p.Annotations.Value + ")"
	}
	return &alert
}

func (a SensuAlert) Send() error {
	if a.Source == "" {
		return fmt.Errorf("empty source: %+v", a)
	}
	devID := a.Source
	if len(devID) > 4 {
		devID = devID[:4]
	}
	b, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("alert marshal: %v", err)
	}
	log.Printf(">>> sensu (%s): %s", devID, b)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *sensuHost, *sensuSocketPort))
	if err != nil {
		return fmt.Errorf("sensu connect: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(50 * time.Millisecond))
	_, err = fmt.Fprintf(conn, "%s", b)
	if err != nil {
		return fmt.Errorf("sensu write: %v", err)
	}
	var buf = make([]byte, 1024)
	if _, err = conn.Read(buf); err != nil {
		return fmt.Errorf("read reply: %v", err)
	}
	log.Printf("<<< sensu (%s): %s", devID, buf)
	return nil
}

func toStatus(severity string) Status {
	switch severity {
	case "ok":
		return OK
	case "warning":
		return WARNING
	case "critical":
		return CRITICAL
	default:
		return UNDEF
	}
}

func (s Status) String() string {
	switch s {
	case OK:
		return "OK"
	case WARNING:
		return "WARN"
	case CRITICAL:
		return "CRIT"
	default:
		return "UNKNOWN"
	}
}
