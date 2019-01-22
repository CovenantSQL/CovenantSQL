package metric

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"html/template"
)

var (
	page = template.Must(template.New("").
		Funcs(template.FuncMap{"path": path, "duration": duration}).
		Parse(`<!DOCTYPE html>
<html lang="us">
<meta charset="utf-8">
<title>Metrics report</title>
<meta name="viewport" content="width=device-width">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; font-family: monospace; font-size: 12px; }
.container {
	max-width: 640px;
	margin: 1em auto;
	display: flex;
	flex-direction: column;
	padding: 0 1em;
}
h1 { text-align: center; }
h2 {
	font-weight: normal;
	text-overflow: ellipsis;
	white-space: nowrap;
	overflow: hidden;
}
.metric {
	padding: 1em 0;
	border-top: 1px solid rgba(0,0,0,0.33);
}
.row {
	display: flex;
	flex-direction: row;
	align-items: center;
	margin: 0.25em 0;
}
.col-1 { flex: 1; }
.col-2 { flex: 2.5; }
.table { width: 100px; border-radius: 2px; border: 1px solid rgba(0,0,0,0.33); }
.table td, .table th { text-align: center; }
.timeline { padding: 0 0.5em; }
path { fill: none; stroke: rgba(0,0,0,0.33); stroke-width: 1; stroke-linecap: round; stroke-linejoin: round; }
path:last-child { stroke: black; }
</style>
<body>
<div class="container">
<div><h1><pre>    __          __
.--------..-----.|  |_ .----.|__|.----..-----.
|        ||  -__||   _||   _||  ||  __||__ --|
|__|__|__||_____||____||__|  |__||____||_____|


</pre></h1></div>
{{ range . }}
	<div class="row metric">
	  <h2 class="col-1">{{ .name }}</h2>
		<div class="col-2">
		{{ if .type }}
			<div class="row">
				{{ template "table" . }}
				<div class="col-1"></div>
			</div>
		{{ else if .interval }}
			<div class="row">{{ template "timeseries" . }}</div>
		{{ else if .metrics}}
			{{ range .metrics }}
				<div class="row">
				{{ template "timeseries" . }}
				</div>
			{{ end }}
		{{ end }}
		</div>
  </div>
{{ end }}
</div>
</body>
</html>
{{ define "table" }}
<table class="table col-1">
	{{ if eq .type "c" }}
		<thead><tr><th>count</th></tr></thead><tbody><tr><td>{{ printf "%.2g" .count }}</td></tr></tbody>
	{{ else if eq .type "g" }}
		<thead><tr><th>mean</th><th>min</th><th>max</th></tr></thead>
		<tbody><tr><td>{{printf "%.2g" .mean}}</td><td>{{printf "%.2g" .min}}</td><td>{{printf "%.2g" .max}}</td></th></tbody>
	{{ else if eq .type "h" }}
		<thead><tr><th>P.50</th><th>P.90</th><th>P.99</th></tr></thead>
		<tbody><tr><td>{{printf "%.2g" .p50}}</td><td>{{printf "%.2g" .p90}}</td><td>{{printf "%.2g" .p99}}</td></tr></tbody>
	{{ end }}
</table>
{{ end }}
{{ define "timeseries" }}
  {{ template "table" .total }}
	<div class="col-1">
		<div class="row">
			<div class="timeline">{{ duration .samples .interval }}</div>
			<svg class="col-1" version="1.1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 20">
			{{ if eq (index (index .samples 0) "type") "c" }}
				{{ range (path .samples "count") }}<path d={{ . }} />{{end}}
			{{ else if eq (index (index .samples 0) "type") "g" }}
				{{ range (path .samples "min" "max" "mean" ) }}<path d={{ . }} />{{end}}
			{{ else if eq (index (index .samples 0) "type") "h" }}
				{{ range (path .samples "p50" "p90" "p99") }}<path d={{ . }} />{{end}}
			{{ end }}
			</svg>
		</div>
	</div>
{{ end }}
`))
)

func path(samples []interface{}, keys ...string) []string {
	var min, max float64
	paths := make([]string, len(keys), len(keys))
	for i := 0; i < len(samples); i++ {
		s := samples[i].(map[string]interface{})
		for _, k := range keys {
			x := s[k].(float64)
			if i == 0 || x < min {
				min = x
			}
			if i == 0 || x > max {
				max = x
			}
		}
	}
	for i := 0; i < len(samples); i++ {
		s := samples[i].(map[string]interface{})
		for j, k := range keys {
			v := s[k].(float64)
			x := float64(i+1) / float64(len(samples))
			y := (v - min) / (max - min)
			if max == min {
				y = 0
			}
			if i == 0 {
				paths[j] = fmt.Sprintf("M%f %f", 0.0, (1-y)*18+1)
			}
			paths[j] += fmt.Sprintf(" L%f %f", x*100, (1-y)*18+1)
		}
	}
	return paths
}

func duration(samples []interface{}, n float64) string {
	n = n * float64(len(samples))
	if n < 60 {
		return fmt.Sprintf("%d sec", int(n))
	} else if n < 60*60 {
		return fmt.Sprintf("%d min", int(n/60))
	} else if n < 24*60*60 {
		return fmt.Sprintf("%d hrs", int(n/60/60))
	}
	return fmt.Sprintf("%d days", int(n/24/60/60))
}

// Handler returns an http.Handler that renders web UI for all provided metrics.
func Handler(snapshot func() map[string]Metric) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type h map[string]interface{}
		metrics := []h{}
		for name, metric := range snapshot() {
			m := h{}
			b, _ := json.Marshal(metric)
			json.Unmarshal(b, &m)
			m["name"] = name
			metrics = append(metrics, m)
		}
		sort.Slice(metrics, func(i, j int) bool {
			n1 := metrics[i]["name"].(string)
			n2 := metrics[j]["name"].(string)
			return strings.Compare(n1, n2) < 0
		})
		page.Execute(w, metrics)
	})
}

// Exposed returns a map of exposed metrics (see expvar package).
func Exposed() map[string]Metric {
	m := map[string]Metric{}
	expvar.Do(func(kv expvar.KeyValue) {
		if metric, ok := kv.Value.(Metric); ok {
			m[kv.Key] = metric
		}
	})
	return m
}
