package caddyawslambda

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"go.uber.org/zap"
)

type fakeLambdaInvoker struct {
	input  *lambda.InvokeInput
	output *lambda.InvokeOutput
	err    error
}

func (f *fakeLambdaInvoker) Invoke(_ context.Context, input *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	f.input = input
	return f.output, f.err
}

func TestInvokeLambdaUsesConfiguredFunctionAndPayload(t *testing.T) {
	fake := &fakeLambdaInvoker{output: &lambda.InvokeOutput{Payload: []byte(`{"ok":true}`)}}
	m := &LambdaMiddleware{
		FunctionName: "test-function",
		timeout:      time.Second,
		log:          zap.NewNop(),
		svc:          fake,
	}

	payload, err := m.invokeLambda(context.Background(), Request{Type: "HTTPJSON-REQ"})
	if err != nil {
		t.Fatalf("invokeLambda() error = %v", err)
	}
	if string(payload) != `{"ok":true}` {
		t.Errorf("payload = %q, want %q", payload, `{"ok":true}`)
	}
	if got := *fake.input.FunctionName; got != "test-function" {
		t.Errorf("function name = %q, want %q", got, "test-function")
	}
	var request Request
	if err := json.Unmarshal(fake.input.Payload, &request); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	if request.Type != "HTTPJSON-REQ" {
		t.Errorf("request type = %q, want HTTPJSON-REQ", request.Type)
	}
}

func TestInvokeLambdaReturnsFunctionError(t *testing.T) {
	functionError := "Unhandled"
	fake := &fakeLambdaInvoker{output: &lambda.InvokeOutput{
		FunctionError: &functionError,
		Payload:       []byte("boom"),
	}}
	m := &LambdaMiddleware{FunctionName: "test-function", timeout: time.Second, log: zap.NewNop(), svc: fake}

	if _, err := m.invokeLambda(context.Background(), struct{}{}); err == nil {
		t.Fatal("invokeLambda() error = nil, want function error")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		middleware    LambdaMiddleware
		wantErrorText string
	}{
		{name: "missing function", wantErrorText: "function must be configured"},
		{
			name:          "unknown event format",
			middleware:    LambdaMiddleware{FunctionName: "test-function", EventFormat: "unknown"},
			wantErrorText: "unsupported event format \"unknown\"",
		},
		{
			name:       "default event format",
			middleware: LambdaMiddleware{FunctionName: "test-function"},
		},
		{
			name:       "api gateway v2",
			middleware: LambdaMiddleware{FunctionName: "test-function", EventFormat: eventFormatAPIGatewayV2},
		},
		{
			name:          "negative max body size",
			middleware:    LambdaMiddleware{FunctionName: "test-function", MaxBodySize: -1},
			wantErrorText: "max_body_size must not be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.middleware.Validate()
			if test.wantErrorText == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErrorText {
				t.Fatalf("Validate() error = %v, want %q", err, test.wantErrorText)
			}
		})
	}
}
