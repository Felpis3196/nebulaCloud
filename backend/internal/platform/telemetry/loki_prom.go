package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LogLine is a dashboard-ready log row.
type LogLine struct {
	Timestamp      string `json:"ts"`
	Level          string `json:"level,omitempty"`
	Service        string `json:"service,omitempty"`
	Message        string `json:"message"`
	CorrelationID  string `json:"correlation_id,omitempty"`
}

// MetricPoint is one sample.
type MetricPoint struct {
	TS    int64   `json:"ts"`
	Value float64 `json:"value"`
}

// MetricSeries for charts.
type MetricSeries struct {
	Name   string        `json:"name"`
	Unit   string        `json:"unit"`
	Points []MetricPoint `json:"points"`
}

// QueryLokiServiceLogs runs LogQL for Docker label stream `service="<id>"`.
func QueryLokiServiceLogs(ctx context.Context, lokiBase, serviceUUID string, window time.Duration, limit int) ([]LogLine, error) {
	if strings.TrimSpace(lokiBase) == "" {
		return nil, nil
	}
	end := time.Now().UTC()
	start := end.Add(-window)

	query := `{service="` + escapeLogQL(serviceUUID) + `"} |= ""`

	endpoint := strings.TrimRight(lokiBase, "/") + "/loki/api/v1/query_range"
	qv := url.Values{}
	qv.Set("query", query)
	qv.Set("limit", strconv.Itoa(limit))
	qv.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	qv.Set("end", strconv.FormatInt(end.UnixNano(), 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+qv.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("loki query: status %d", res.StatusCode)
	}

	var envelope struct {
		Data struct {
			Result []struct {
				Values [][]string `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	var lines []LogLine
	for _, r := range envelope.Data.Result {
		for _, pair := range r.Values {
			if len(pair) < 2 {
				continue
			}
			ns, msg := pair[0], pair[1]
			tsNanos, err := strconv.ParseInt(ns, 10, 64)
			if err != nil {
				continue
			}
			lines = append(lines, LogLine{
				Timestamp: time.Unix(0, tsNanos).UTC().Format(time.RFC3339Nano),
				Message:   msg,
				Level:   "info",
			})
		}
	}
	return lines, nil
}

func escapeLogQL(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// QueryPrometheusRange returns points for expr (best-effort).
func QueryPrometheusRange(ctx context.Context, promBase, expr string, window, step time.Duration) ([]MetricPoint, []string, error) {
	if strings.TrimSpace(promBase) == "" {
		return nil, nil, fmt.Errorf("no prometheus URL")
	}
	end := time.Now().UTC().Unix()
	start := time.Now().Add(-window).UTC().Unix()

	u := strings.TrimSuffix(promBase, "/") + "/api/v1/query_range"
	form := url.Values{}
	form.Set("query", expr)
	form.Set("start", strconv.FormatInt(start, 10))
	form.Set("end", strconv.FormatInt(end, 10))
	form.Set("step", fmt.Sprintf("%.0f", step.Seconds()))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	var out struct {
		Data struct {
			Result []struct {
				Values [][]any `json:"values"`
			} `json:"result"`
			ResultType string `json:"resultType"`
		} `json:"data"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil || out.Status != "success" || len(out.Data.Result) == 0 {
		return nil, nil, err
	}
	var pts []MetricPoint
	for _, row := range out.Data.Result[0].Values {
		if len(row) < 2 {
			continue
		}
		tf, ok1 := row[0].(float64)
		vf, ok2 := row[1].(string)
		if !ok1 {
			ts, _ := row[0].(json.Number)
			tfloat, _ := ts.Float64()
			tf = tfloat
		}
		var val float64
		if ok2 {
			val, _ = strconv.ParseFloat(vf, 64)
		} else if f, ok := row[1].(float64); ok {
			val = f
		}
		pts = append(pts, MetricPoint{TS: int64(tf * 1000), Value: val})
	}
	return pts, []string{}, nil
}
