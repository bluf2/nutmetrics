package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var version = "dev"

func main() {
	server := flag.String("nut.server", "localhost:3493", "NUT server, host or host:port")
	ups := flag.String("nut.ups", "", "UPS name; empty means the first UPS the server lists")
	timeout := flag.Duration("nut.timeout", 5*time.Second, "timeout for one scrape of the NUT server")
	listen := flag.String("web.listen-address", ":9199", "address to listen on")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	h := &handler{addr: withPort(*server), ups: *ups, timeout: *timeout}
	mux := http.NewServeMux()
	mux.Handle("/metrics", h)
	mux.Handle("/ups_metrics", h) // druggeri/nut_exporter path; ?ups=NAME overrides --nut.ups
	log.Printf("nutmetrics %s: %s -> %s", version, *listen, h.addr)
	log.Fatal(http.ListenAndServe(*listen, mux))
}
