package caddyawslambda

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequestForFormatAPIGatewayV2(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "http://example.test/items?tag=one&tag=two", strings.NewReader("\x00\x01"))
	r.RemoteAddr = "192.0.2.1:1234"
	r.Header.Add("X-Test", "one")
	r.Header.Add("X-Test", "two")
	r.Header.Set("Content-Type", "application/octet-stream")
	r.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

	payload, err := newRequestForFormat(r, eventFormatAPIGatewayV2)
	if err != nil {
		t.Fatalf("newRequestForFormat() error = %v", err)
	}

	event, ok := payload.(*APIGatewayV2Request)
	if !ok {
		t.Fatalf("payload type = %T, want *APIGatewayV2Request", payload)
	}
	if event.Version != "2.0" {
		t.Errorf("Version = %q, want %q", event.Version, "2.0")
	}
	if event.RawPath != "/items" || event.RawQueryString != "tag=one&tag=two" {
		t.Errorf("path = %q?%q, want /items?tag=one&tag=two", event.RawPath, event.RawQueryString)
	}
	if event.Headers["x-test"] != "one,two" {
		t.Errorf("x-test = %q, want %q", event.Headers["x-test"], "one,two")
	}
	if event.QueryStringParameters["tag"] != "one,two" {
		t.Errorf("tag = %q, want %q", event.QueryStringParameters["tag"], "one,two")
	}
	if len(event.Cookies) != 1 || event.Cookies[0] != "session=abc" {
		t.Errorf("Cookies = %#v, want [session=abc]", event.Cookies)
	}
	if event.RequestContext.HTTP.SourceIP != "192.0.2.1" {
		t.Errorf("SourceIP = %q, want %q", event.RequestContext.HTTP.SourceIP, "192.0.2.1")
	}
	if event.Body != base64.StdEncoding.EncodeToString([]byte("\x00\x01")) || !event.IsBase64Encoded {
		t.Errorf("binary body = %q, encoded = %v", event.Body, event.IsBase64Encoded)
	}
}

func TestNewRequestForFormatDefaultsToHTTPJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("body"))
	m := &LambdaMiddleware{}

	payload, err := newRequestForFormat(r, m.eventFormat())
	if err != nil {
		t.Fatalf("newRequestForFormat() error = %v", err)
	}

	request, ok := payload.(*Request)
	if !ok {
		t.Fatalf("payload type = %T, want *Request", payload)
	}
	if request.Type != "HTTPJSON-REQ" || request.Body != "body" {
		t.Errorf("request = %#v, want HTTPJSON-REQ with body", request)
	}
}

func TestNewRequestForFormatRejectsUnknownFormat(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := newRequestForFormat(r, "unknown"); err == nil {
		t.Fatal("newRequestForFormat() error = nil, want error")
	}
}
