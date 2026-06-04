package main

import (
	"strings"
	"testing"
)

func TestParseHTTPRequestsTotal(t *testing.T) {
	input := strings.NewReader(`# HELP taxi_http_requests_total Total HTTP requests handled by taxi API.
# TYPE taxi_http_requests_total counter
taxi_http_requests_total{method="GET",path="/api/v1/health",status="200"} 3
taxi_http_requests_total{method="POST",path="/api/v1/auth/login",status="200"} 2
`)

	total, err := parseHTTPRequestsTotal(input)
	if err != nil {
		t.Fatalf("parse metrics: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %v, want 5", total)
	}
}

func TestFitLineTruncatesByRuneWidth(t *testing.T) {
	result := fitLine("abcdef", 4)
	if result != "abc>" {
		t.Fatalf("result = %q, want %q", result, "abc>")
	}
}

func TestPowerShellSingleQuotedEscapesQuotes(t *testing.T) {
	result := powerShellSingleQuoted(`q:\taxi-platform's`)
	if result != `'q:\taxi-platform''s'` {
		t.Fatalf("result = %q", result)
	}
}

func TestJoinPowerShellArguments(t *testing.T) {
	result := joinPowerShellArguments([]string{"monitor", "--metrics-url", "http://127.0.0.1:8080/metrics"})
	expected := `'monitor' '--metrics-url' 'http://127.0.0.1:8080/metrics'`
	if result != expected {
		t.Fatalf("result = %q, want %q", result, expected)
	}
}
