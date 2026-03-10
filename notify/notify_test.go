package notify

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabioconcina/arpdvark/scanner"
)

func TestSendEmpty(t *testing.T) {
	if err := Send("http://example.com", nil); err != nil {
		t.Fatalf("expected nil error for empty devices, got %v", err)
	}
	if err := Send("", []scanner.Device{{}}); err != nil {
		t.Fatalf("expected nil error for empty URL, got %v", err)
	}
}

func TestSendPost(t *testing.T) {
	var gotBody string
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.50"),
			MAC:      net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			Vendor:   "Acme Corp",
			Hostname: "myhost",
		},
	}

	if err := Send(srv.URL, devices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotContentType != "text/plain" {
		t.Errorf("content-type = %q, want text/plain", gotContentType)
	}

	want := "New device: 192.168.1.50 (aa:bb:cc:dd:ee:ff) Acme Corp myhost"
	if gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
}

func TestSendServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	devices := []scanner.Device{
		{IP: net.ParseIP("10.0.0.1"), MAC: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}},
	}

	err := Send(srv.URL, devices)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
