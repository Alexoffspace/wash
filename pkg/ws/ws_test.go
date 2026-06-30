package ws

import (
	"net/http/httptest"
	"testing"
)

func TestAllowedOriginAllow0TrueSameHost(t *testing.T) {
	r := httptest.NewRequest("GET", "http://fishnet.vita.local:9091/ws", nil)
	r.Header.Set("Origin", "http://fishnet.vita.local:9091")

	if !isAllowedOrigin(r, true) {
		t.Fatal("expected same-host origin to be allowed when allow0=true")
	}
}

func TestAllowedOriginAllow0TrueRejectsCrossSite(t *testing.T) {
	r := httptest.NewRequest("GET", "http://fishnet.vita.local:9091/ws", nil)
	r.Header.Set("Origin", "http://evil.example:9091")

	if isAllowedOrigin(r, true) {
		t.Fatal("expected cross-site origin to be rejected when allow0=true")
	}
}

func TestAllowedOriginAllow0FalseAllowsLoopback(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		origin string
	}{
		{name: "localhost", url: "http://localhost:9091/ws", origin: "http://localhost:9091"},
		{name: "ipv4 loopback", url: "http://127.0.0.1:9091/ws", origin: "http://127.0.0.1:9091"},
		{name: "ipv6 loopback", url: "http://[::1]:9091/ws", origin: "http://[::1]:9091"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.url, nil)
			r.Header.Set("Origin", tt.origin)

			if !isAllowedOrigin(r, false) {
				t.Fatalf("expected %s origin to be allowed when allow0=false", tt.name)
			}
		})
	}
}

func TestAllowedOriginAllow0FalseRejectsNonLoopback(t *testing.T) {
	r := httptest.NewRequest("GET", "http://fishnet.vita.local:9091/ws", nil)
	r.Header.Set("Origin", "http://fishnet.vita.local:9091")

	if isAllowedOrigin(r, false) {
		t.Fatal("expected non-loopback origin to be rejected when allow0=false")
	}
}
