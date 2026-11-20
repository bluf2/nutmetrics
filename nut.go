package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Var is one NUT variable of a UPS, e.g. Name "battery.charge", Value "100".
type Var struct {
	Name  string
	Value string
}

// Client speaks the NUT network protocol (TCP, port 3493) one command at a time.
type Client struct {
	conn net.Conn
	r    *bufio.Reader
}

// Dial connects to a NUT server; timeout covers the connect and every later read and write.
func Dial(addr string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	return &Client{conn: conn, r: bufio.NewReader(conn)}, nil
}

func (c *Client) Close() error {
	_, _ = fmt.Fprint(c.conn, "LOGOUT\n")
	return c.conn.Close()
}

// ListUPS returns the names of the UPSes the server knows.
func (c *Client) ListUPS() ([]string, error) {
	lines, err := c.list("LIST UPS")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) >= 2 && f[0] == "UPS" {
			names = append(names, f[1])
		}
	}
	return names, nil
}

// ListVars returns all variables of one UPS in server order.
func (c *Client) ListVars(ups string) ([]Var, error) {
	lines, err := c.list("LIST VAR " + ups)
	if err != nil {
		return nil, err
	}
	var vars []Var
	for _, l := range lines {
		if v, ok := parseVar(l, ups); ok {
			vars = append(vars, v)
		}
	}
	return vars, nil
}

// list sends cmd and returns the lines between "BEGIN <cmd>" and "END <cmd>".
func (c *Client) list(cmd string) ([]string, error) {
	if _, err := fmt.Fprintf(c.conn, "%s\n", cmd); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, fmt.Errorf("%s: %w", cmd, err)
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case strings.HasPrefix(line, "ERR "):
			return nil, fmt.Errorf("%s: NUT server said %s", cmd, line)
		case line == "BEGIN "+cmd:
		case line == "END "+cmd:
			return lines, nil
		default:
			lines = append(lines, line)
		}
	}
}

// parseVar reads `VAR <ups> <name> "<value>"`, undoing the \" and \\ escapes.
func parseVar(line, ups string) (Var, bool) {
	prefix := "VAR " + ups + " "
	if !strings.HasPrefix(line, prefix) {
		return Var{}, false
	}
	rest := line[len(prefix):]
	i := strings.IndexByte(rest, ' ')
	if i < 0 {
		return Var{}, false
	}
	raw := strings.TrimSpace(rest[i+1:])
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return Var{}, false
	}
	raw = raw[1 : len(raw)-1]
	val := strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(raw)
	return Var{Name: rest[:i], Value: val}, true
}
