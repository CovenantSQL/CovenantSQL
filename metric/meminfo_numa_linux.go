// +build !nomeminfo_numa

package metric

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	memInfoNumaSubsystem = "memory_numa"
)

var meminfoNodeRE = regexp.MustCompile(`.*devices/system/node/node([0-9]*)`)

type meminfoMetric struct {
	metricName string
	metricType prometheus.ValueType
	numaNode   string
	value      float64
}

type meminfoNumaCollector struct {
	metricDescs map[string]*prometheus.Desc
}

func parseMemInfoNuma(r io.Reader) ([]meminfoMetric, error) {
	var (
		memInfo []meminfoMetric
		scanner = bufio.NewScanner(r)
		re      = regexp.MustCompile(`\((.*)\)`)
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)

		fv, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value in meminfo: %s", err)
		}
		switch l := len(parts); {
		case l == 4: // no unit
		case l == 5 && parts[4] == "kB": // has unit
			fv *= 1024
		default:
			return nil, fmt.Errorf("invalid line in meminfo: %s", line)
		}
		metric := strings.TrimRight(parts[2], ":")

		// Active(anon) -> Active_anon
		metric = re.ReplaceAllString(metric, "_${1}")
		memInfo = append(memInfo, meminfoMetric{metric, prometheus.GaugeValue, parts[1], fv})
	}

	return memInfo, scanner.Err()
}

func parseMemInfoNumaStat(r io.Reader, nodeNumber string) ([]meminfoMetric, error) {
	var (
		numaStat []meminfoMetric
		scanner  = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line scan did not return 2 fields: %s", line)
		}

		fv, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value in numastat: %s", err)
		}

		numaStat = append(numaStat, meminfoMetric{parts[0] + "_total", prometheus.CounterValue, nodeNumber, fv})
	}
	return numaStat, scanner.Err()
}
