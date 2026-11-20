package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type handler struct {
	addr    string
	ups     string
	timeout time.Duration
}

func withPort(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(s, "3493")
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	body, err := h.scrape(r.URL.Query().Get("ups"))
	var b strings.Builder
	up := 1
	if err != nil {
		up = 0
	}
	b.WriteString("# TYPE nutmetrics_up gauge\n")
	fmt.Fprintf(&b, "nutmetrics_up %d\n", up)
	b.WriteString("# TYPE nutmetrics_scrape_duration_seconds gauge\n")
	fmt.Fprintf(&b, "nutmetrics_scrape_duration_seconds %g\n", time.Since(start).Seconds())
	b.WriteString(body)
	if err != nil {
		fmt.Fprintf(&b, "# error: %s\n", strings.ReplaceAll(err.Error(), "\n", " "))
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) // Prometheus then sees the target as down
	}
	_, _ = w.Write([]byte(b.String()))
}

func (h *handler) scrape(upsParam string) (string, error) {
	c, err := Dial(h.addr, h.timeout)
	if err != nil {
		return "", err
	}
	defer c.Close()
	name := upsParam
	if name == "" {
		name = h.ups
	}
	if name == "" {
		names, err := c.ListUPS()
		if err != nil {
			return "", err
		}
		if len(names) == 0 {
			return "", fmt.Errorf("the NUT server lists no UPS")
		}
		name = names[0]
	}
	vars, err := c.ListVars(name)
	if err != nil {
		return "", err
	}
	return Render(vars), nil
}
