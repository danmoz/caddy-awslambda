package caddyawslambda

import (
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestUnmarshalCaddyfileEndpoint(t *testing.T) {
	m := &LambdaMiddleware{}
	if err := m.UnmarshalCaddyfile(caddyfile.NewTestDispenser(`
		awslambda {
			function test-function
			endpoint http://127.0.0.1:3001
			region us-east-1
			access_key_id test
			secret_access_key secret
			session_token token
			timeout 5s
		}
	`)); err != nil {
		t.Fatalf("UnmarshalCaddyfile() error = %v", err)
	}

	if m.FunctionName != "test-function" {
		t.Errorf("FunctionName = %q, want %q", m.FunctionName, "test-function")
	}
	if m.Endpoint != "http://127.0.0.1:3001" {
		t.Errorf("Endpoint = %q, want %q", m.Endpoint, "http://127.0.0.1:3001")
	}
	if m.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", m.Region, "us-east-1")
	}
	if m.AccessKeyID != "test" || m.SecretAccessKey != "secret" || m.SessionToken != "token" {
		t.Errorf("local credentials = %q/%q/%q", m.AccessKeyID, m.SecretAccessKey, m.SessionToken)
	}
	if m.Timeout != "5s" {
		t.Errorf("Timeout = %q, want %q", m.Timeout, "5s")
	}
}
