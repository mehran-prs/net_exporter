// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nonetstat
// +build !nonetstat

package collector

import (
	"bufio"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

// TODO: rename this file to netstat_linux.go because this type of netstat is just available on linux os.

var socketStates = [...]string{
	"UNKNOWN",
	"ESTABLISHED",
	"SYN_SENT",
	"SYN_RECV",
	"FIN_WAIT1",
	"FIN_WAIT2",
	"TIME_WAIT",
	"_close", // CLOSE
	"CLOSE_WAIT",
	"LAST_ACK",
	"LISTEN",
	"CLOSING",
}

const (
	netStatsSubsystem = "netstat"
)

var (
	asnFilePath = kingpin.Flag("collector.netstat.asn_file", "ASN CSV file's path").Default("/etc/net_exporter/asn_db.csv").String()
)

type netStatCollector struct {
	asnRecords []*asnRecord
	logger     log.Logger
	socketDesc *prometheus.Desc
}

type asnRecord struct {
	IPNet *net.IPNet
	Name  string // Autonomous system (AS) name.
}

func init() {
	registerCollector("connstat", defaultEnabled, NewNetStatCollector)
}

// NewNetStatCollector takes and returns
// a new Collector exposing network stats.
func NewNetStatCollector(logger log.Logger) (Collector, error) {
	f, err := os.Open(*asnFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	asnRecords, err := parseAsnRecords(f)
	if err != nil {
		return nil, err
	}

	return &netStatCollector{
		asnRecords: asnRecords,
		logger:     logger,
		socketDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, netStatsSubsystem, "sockets"),
			"Current os sockets",
			[]string{"asn", "state", "ipv"}, nil,
		),
	}, nil
}

func (c *netStatCollector) Update(ch chan<- prometheus.Metric) error {
	tcpStats, err := getSocketStats(procFilePath("net/tcp"), c.asnRecords)
	if err != nil {
		return fmt.Errorf("couldn't get netstats: %w", err)
	}
	tcp6Stats, err := getSocketStats(procFilePath("net/tcp6"), c.asnRecords)
	if err != nil {
		return fmt.Errorf("couldn't get SNMP stats: %w", err)
	}

	for asn, asnStats := range tcpStats {
		for state, value := range asnStats {
			ch <- prometheus.MustNewConstMetric(c.socketDesc, prometheus.GaugeValue, float64(value), asn, state, "4")
		}
	}

	for asn, asnStats := range tcp6Stats {
		for state, value := range asnStats {
			ch <- prometheus.MustNewConstMetric(c.socketDesc, prometheus.GaugeValue, float64(value), asn, state, "6")
		}
	}

	return nil
}

// getSocketStats returns socket stats from /proc/net/tcp(6)|udp(6) files.
func getSocketStats(fileName string, asns []*asnRecord) (map[string]map[string]uint64, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseSocketStats(file, asns)
}

func parseSocketStats(r io.Reader, asns []*asnRecord) (map[string]map[string]uint64, error) {
	// First key is asn name. second key is connection state,
	// value is connections count.
	netStats := map[string]map[string]uint64{}
	scanner := bufio.NewScanner(r)

	// Skip header
	scanner.Scan() // Skip the header

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}

		fields := strings.Fields(line)
		if len(fields) < 12 {
			return nil, fmt.Errorf("netstat: not enough fields: %v, %v", len(fields), fields)
		}

		//localIP, localPort, err := parseAddr(fields[1])
		//if err != nil {
		//	return nil, fmt.Errorf("can not parse local ip and port address in the file: %w", err)
		//}

		remoteIP, _, err := parseAddr(fields[2])
		if err != nil {
			return nil, fmt.Errorf("can not parse remote ip and port ddress in the file: %w", err)
		}

		u, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			return nil, err
		}

		state := socketStates[u]
		var asName = "_other"
		if asn := findASN(asns, *remoteIP); asn != nil {
			asName = asn.Name
		}

		if netStats[asName] == nil {
			netStats[asName] = map[string]uint64{}
		}
		netStats[asName][state]++
	}

	return netStats, scanner.Err()
}

func parseAsnRecords(r io.Reader) ([]*asnRecord, error) {
	l := make([]*asnRecord, 0)
	reader := csv.NewReader(r)

	// read more about the asn DB file's format [here](https://lite.ip2location.com/database-asn)
	for record, err := reader.Read(); err == nil; record, err = reader.Read() {
		_, ipnet, err := net.ParseCIDR(record[2])
		if err != nil {
			return nil, fmt.Errorf("can not parse ip value: %s: %w", record[2], err)
		}
		l = append(l, &asnRecord{
			IPNet: ipnet,
			Name:  record[4],
		})
	}

	return l, nil
}

func findASN(l []*asnRecord, ip net.IP) *asnRecord {
	for _, r := range l {
		if r.IPNet.Contains(ip) {
			return r
		}
	}
	return nil
}

//--------------------------------
// Got socket file parser source from
// [cakturk/go-netstat](https://github.com/cakturk/go-netstat)
// repo.
//--------------------------------

func parseIPv4(s string) (net.IP, error) {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	ip := make(net.IP, net.IPv4len)
	binary.LittleEndian.PutUint32(ip, uint32(v))
	return ip, nil
}

func parseIPv6(s string) (net.IP, error) {
	ip := make(net.IP, net.IPv6len)
	const grpLen = 4
	i, j := 0, 4
	for len(s) != 0 {
		grp := s[0:8]
		u, err := strconv.ParseUint(grp, 16, 32)
		binary.LittleEndian.PutUint32(ip[i:j], uint32(u))
		if err != nil {
			return nil, err
		}
		i, j = i+grpLen, j+grpLen
		s = s[8:]
	}
	return ip, nil
}

// parseAddr parses ip:port address in the /proc/net/tcp, /proc/net/udp files.
func parseAddr(s string) (*net.IP, uint16, error) {
	fields := strings.Split(s, ":")
	if len(fields) < 2 {
		return nil, 0, fmt.Errorf("netstat: not enough fields: %v", s)
	}
	var ip net.IP
	var err error
	switch len(fields[0]) {
	case 8: // IPv4 string length
		ip, err = parseIPv4(fields[0])
	case 32: // IPv6 string length
		ip, err = parseIPv6(fields[0])
	default:
		err = fmt.Errorf("netstat: bad formatted string: %v", fields[0])
	}
	if err != nil {
		return nil, 0, err
	}
	v, err := strconv.ParseUint(fields[1], 16, 16)
	if err != nil {
		return nil, 0, err
	}
	return &ip, uint16(v), nil
}
