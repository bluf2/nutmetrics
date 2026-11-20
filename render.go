package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const prefix = "network_ups_tools_"

// statusFlags are the documented ups.status tokens; each is exported as 0 or 1 so alerts see a flag go back to 0.
var statusFlags = []string{"BOOST", "BYPASS", "CAL", "CHRG", "DISCHRG", "FSD", "HB", "LB", "OB", "OFF", "OL", "OVER", "RB", "SD", "TRIM"}

var (
	numberRE  = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	invalidRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func promName(nutName string) string { return invalidRE.ReplaceAllString(nutName, "_") }

// Render turns the variables of one UPS into the Prometheus text format:
//   - numeric variables as network_ups_tools_<name> (same names as druggeri/nut_exporter),
//   - enabled/disabled variables as 1/0 (and as an nut_ups_info label too),
//   - ups.status as one network_ups_tools_ups_status{flag="..."} per flag,
//   - the remaining text variables as labels of nut_ups_info. ups.status is left out, because a label that changes
//     with its value would make a new series on every change.
func Render(vars []Var) string {
	sorted := append([]Var(nil), vars...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var b strings.Builder
	var info []string
	for _, v := range sorted {
		val := strings.TrimSpace(v.Value)
		switch {
		case v.Name == "ups.status":
			writeStatus(&b, val)
		case numberRE.MatchString(val):
			f, _ := strconv.ParseFloat(val, 64)
			fmt.Fprintf(&b, "%s%s %s\n", prefix, promName(v.Name), strconv.FormatFloat(f, 'g', -1, 64))
		case val == "enabled" || val == "disabled":
			n := 0
			if val == "enabled" {
				n = 1
			}
			fmt.Fprintf(&b, "%s%s %d\n", prefix, promName(v.Name), n)
			info = append(info, fmt.Sprintf(`%s="%s"`, promName(v.Name), val)) // keeps the text in nut_ups_info too
		case val != "":
			info = append(info, fmt.Sprintf(`%s="%s"`, promName(v.Name), escapeLabel(val)))
		}
	}
	if len(info) > 0 {
		fmt.Fprintf(&b, "# TYPE nut_ups_info gauge\nnut_ups_info{%s} 1\n", strings.Join(info, ","))
	}
	return b.String()
}

func writeStatus(b *strings.Builder, status string) {
	set := map[string]bool{}
	for _, f := range strings.Fields(status) {
		set[f] = true
	}
	flags := append([]string(nil), statusFlags...)
	for f := range set {
		if !contains(flags, f) {
			flags = append(flags, f)
		}
	}
	sort.Strings(flags)
	for _, f := range flags {
		n := 0
		if set[f] {
			n = 1
		}
		fmt.Fprintf(b, "%sups_status{flag=\"%s\"} %d\n", prefix, escapeLabel(f), n)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func escapeLabel(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", " ").Replace(v)
}
