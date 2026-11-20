package main

import (
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeNUT serves the captured DSM session for any LIST command and "ERR UNKNOWN-UPS" for other UPS names.
func fakeNUT(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/dsm-cyberpower.txt")
	if err != nil {
		t.Fatal(err)
	}
	sections := map[string]string{}
	var cur string
	var buf strings.Builder
	for _, l := range strings.SplitAfter(string(data), "\n") {
		if strings.HasPrefix(l, "BEGIN ") {
			cur, buf = strings.TrimSpace(strings.TrimPrefix(l, "BEGIN ")), strings.Builder{}
		}
		if cur != "" {
			buf.WriteString(l)
		}
		if strings.HasPrefix(l, "END ") {
			sections[cur] = buf.String()
			cur = ""
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				buf := make([]byte, 256)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					cmd := strings.TrimSpace(string(buf[:n]))
					if cmd == "LOGOUT" {
						return
					}
					if s, ok := sections[strings.TrimPrefix(cmd, "")]; ok {
						c.Write([]byte(s))
					} else {
						c.Write([]byte("ERR UNKNOWN-UPS\n"))
					}
				}
			}()
		}
	}()
	return ln.Addr().String()
}

func scrape(t *testing.T, h *handler, path string) (int, string) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec.Code, rec.Body.String()
}

func TestScrapeMatchesDruggeriNames(t *testing.T) {
	h := &handler{addr: fakeNUT(t), timeout: 2 * time.Second}
	code, body := scrape(t, h, "/metrics")
	if code != 200 {
		t.Fatalf("code %d\n%s", code, body)
	}
	for _, want := range []string{
		"nutmetrics_up 1\n",
		"network_ups_tools_battery_charge 100\n",
		"network_ups_tools_battery_voltage 13.7\n",
		"network_ups_tools_ups_productid 501\n",   // "0501" is a number
		"network_ups_tools_input_frequency 496\n", // "496.0" prints without the .0
		"network_ups_tools_ups_timer_shutdown -60\n",
		"network_ups_tools_ups_beeper_status 1\n",
		`network_ups_tools_ups_status{flag="OL"} 1`,
		`network_ups_tools_ups_status{flag="OB"} 0`,
		`battery_type="PbAcid"`,
		`ups_beeper_status="enabled"`,
		`driver_version_data="CyberPower HID 0.6"`,
		`ups_test_result="No test initiated"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, `ups_status="`) {
		t.Error("ups.status must not become an info label")
	}
	if strings.Contains(body, "device_serial") {
		t.Error("blank serial must be skipped")
	}
}

func TestUpsParamAndUnknownUPS(t *testing.T) {
	h := &handler{addr: fakeNUT(t), timeout: 2 * time.Second}
	if code, body := scrape(t, h, "/ups_metrics?ups=ups"); code != 200 || !strings.Contains(body, "nutmetrics_up 1") {
		t.Errorf("ups=ups: %d\n%s", code, body)
	}
	code, body := scrape(t, h, "/ups_metrics?ups=nope")
	if code != 500 || !strings.Contains(body, "nutmetrics_up 0") || !strings.Contains(body, "UNKNOWN-UPS") {
		t.Errorf("ups=nope: %d\n%s", code, body)
	}
}

func TestUnreachableServerIsDown(t *testing.T) {
	h := &handler{addr: "127.0.0.1:1", timeout: time.Second}
	code, body := scrape(t, h, "/metrics")
	if code != 500 || !strings.Contains(body, "nutmetrics_up 0") {
		t.Errorf("%d\n%s", code, body)
	}
}

func TestStatusFlagsAndEscaping(t *testing.T) {
	out := Render([]Var{
		{"ups.status", "OB DISCHRG LB"},
		{"ups.mfr", `A"B\C`},
		{"ups.mystery", "1.5"},
	})
	for _, want := range []string{
		`ups_status{flag="OB"} 1`, `ups_status{flag="DISCHRG"} 1`, `ups_status{flag="LB"} 1`,
		`ups_status{flag="OL"} 0`, `ups_mfr="A\"B\\C"`, "network_ups_tools_ups_mystery 1.5\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if !strings.Contains(Render([]Var{{"ups.status", "WEIRD"}}), `flag="WEIRD"} 1`) {
		t.Error("unknown flags must be exported")
	}
}

func TestParseVarUnescapes(t *testing.T) {
	v, ok := parseVar(`VAR ups ups.mfr "a \"b\" \\c"`, "ups")
	if !ok || v.Name != "ups.mfr" || v.Value != `a "b" \c` {
		t.Errorf("%+v %v", v, ok)
	}
}
