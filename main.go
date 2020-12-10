package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/vma/getopt"
	"github.com/vma/httplogger"
)

const MaxOpenConns = 50

var (
	// Revision is the git revision, set at compilation
	Revision string

	// Build is the build time, set at compilation
	Build string

	// Branch is the git branch, set at compilation
	Branch string

	port            = getopt.Int16Long("port", 'p', 8086, "web server listen port", "port")
	debug           = getopt.BoolLong("debug", 'd', "enable debug mode")
	dumpBody        = getopt.BoolLong("dump-body", 'D', "dumps received alert body")
	sensuHost       = getopt.StringLong("sensu-host", 0, "localhost", "sensu socket server host")
	sensuSocketPort = getopt.Int16Long("sensu-client-port", 0, 3030, "sensu socket server port (to send events)")
	sensuApiPort    = getopt.Int16Long("sensu-api-port", 0, 4567, "sensu api server port (to update clients)")
	connSlots       = make(chan struct{}, MaxOpenConns)
	showVersion     = getopt.BoolLong("version", 'v', "Prints version and build date")
)

func main() {
	getopt.SetParameters("")
	getopt.Parse()

	if *showVersion {
		fmt.Printf("Revision:%s Branch:%s Build:%s\n", Revision, Branch, Build)
		return
	}

	log.SetFlags(log.Ltime | log.Lmicroseconds) // to be used with system logger
	http.HandleFunc("/alert", HandleAlert)
	logger := httplogger.CommonLogger(os.Stderr)
	log.Printf("using sensu socket server at %s:%d", *sensuHost, *sensuSocketPort)
	log.Printf("using sensu api server at %s:%d", *sensuHost, *sensuApiPort)
	log.Printf("starting web server on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), logger(http.DefaultServeMux)))
}

func HandleAlert(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERR: read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	if *dumpBody {
		log.Printf(">> %s", b)
	}
	var promAlerts PromAlerts
	if err := json.Unmarshal(b, &promAlerts); err != nil {
		log.Printf("ERR: decode prom alert: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	go func(promAlerts PromAlerts) {
		dbg("prom alerts: %+v", promAlerts)
		for _, a := range promAlerts.Alerts {
			cli := NewClientFromAlert(a)
			if cli != nil {
				//dbg("** cli update: getting conn slot (%d used)", len(connSlots))
				connSlots <- struct{}{}
				err := cli.Update()
				if err != nil {
					log.Printf("ERR: update sensu client: %v", err)
				}
				//dbg("++ cli update: done, releasing conn slot")
				<-connSlots
			}

			alert := a.ToSensu()
			if alert != nil {
				//dbg("** send alert: getting conn slot (%d used)", len(connSlots))
				connSlots <- struct{}{}
				err := alert.Send()
				if err != nil {
					log.Printf("ERR: send sensu alert: %v", err)
				}
				//dbg("++ send alert: done, releasing conn slot")
				<-connSlots
			}
		}
	}(promAlerts)
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Connection", "close")
}

func dbg(fmt string, v ...interface{}) {
	if *debug {
		log.Printf(fmt, v...)
	}
}
