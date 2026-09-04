package caddyawslambda

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

type lambdaInvoker interface {
	Invoke(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

var (
	_ caddy.Module                = (*LambdaMiddleware)(nil)
	_ caddy.Provisioner           = (*LambdaMiddleware)(nil)
	_ caddy.Validator             = (*LambdaMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*LambdaMiddleware)(nil)
)

func init() {
	caddy.RegisterModule(&LambdaMiddleware{})
}

// LambdaMiddleware implements an HTTP handler that invokes a Lambda function.
type LambdaMiddleware struct {
	FunctionName    string `json:"function,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
	EventFormat     string `json:"event_format,omitempty"`
	Timeout         string `json:"timeout,omitempty"`
	MaxBodySize     int64  `json:"max_body_size,omitempty"`

	timeout time.Duration
	log     *zap.Logger
	svc     lambdaInvoker
}

// CaddyModule returns the Caddy module information.
func (*LambdaMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.awslambda",
		New: func() caddy.Module { return &LambdaMiddleware{} },
	}
}

// Provision implements caddy.Provisioner.
func (m *LambdaMiddleware) Provision(ctx caddy.Context) error {
	m.log = ctx.Logger(m)

	if m.Timeout == "" {
		m.Timeout = "10s"
	}

	dur, err := time.ParseDuration(m.Timeout)
	if err != nil {
		return fmt.Errorf("invalid value for timeout: %w", err)
	}
	m.timeout = dur

	if (m.AccessKeyID == "") != (m.SecretAccessKey == "") {
		return errors.New("access_key_id and secret_access_key must be configured together")
	}
	configOptions := []func(*config.LoadOptions) error{}
	if m.Region != "" {
		configOptions = append(configOptions, config.WithRegion(m.Region))
	}
	if m.AccessKeyID != "" {
		configOptions = append(configOptions, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			m.AccessKeyID, m.SecretAccessKey, m.SessionToken,
		)))
	}
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	if m.Endpoint == "" {
		m.svc = lambda.NewFromConfig(cfg)
	} else {
		m.svc = lambda.NewFromConfig(cfg, func(options *lambda.Options) {
			options.BaseEndpoint = &m.Endpoint
		})
	}
	return nil
}

// Validate implements caddy.Validator.
func (m *LambdaMiddleware) Validate() error {
	if m.MaxBodySize < 0 {
		return errors.New("max_body_size must not be negative")
	}
	if m.FunctionName == "" {
		return errors.New("function must be configured")
	}

	switch m.eventFormat() {
	case eventFormatHTTPJSON, eventFormatAPIGatewayV2:
		return nil
	default:
		return fmt.Errorf("unsupported event format %q", m.EventFormat)
	}
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m *LambdaMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	req, err := newRequestForFormat(r, m.eventFormat(), m.MaxBodySize)
	if err != nil {
		return err
	}

	resp, err := m.invokeLambda(r.Context(), req)

	if err != nil {
		return err
	}

	// Unpack the reply JSON
	reply, err := parseReply(resp, m.eventFormat())
	if err != nil {
		return err
	}
	if err := validateReply(reply); err != nil {
		return err
	}

	// Optionally decode the response body before writing any response data.
	var bodyBytes []byte
	if reply.BodyEncoding == "base64" && reply.Body != "" {
		bodyBytes, err = base64.StdEncoding.DecodeString(reply.Body)
		if err != nil {
			return err
		}
	} else {
		bodyBytes = []byte(reply.Body)
	}

	// Write the response HTTP headers
	for k, vals := range reply.Meta.Headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	// Default the Content-Type to application/json if not provided on reply
	if w.Header().Get("content-type") == "" {
		w.Header().Set("content-type", "application/json")
	}
	if reply.Meta.Status <= 0 {
		reply.Meta.Status = http.StatusOK
	}

	w.WriteHeader(reply.Meta.Status)

	// Write the response body
	_, err = w.Write(bodyBytes)
	if err != nil || reply.Meta.Status >= 400 {
		return err
	}

	return nil
}

func (m *LambdaMiddleware) invokeLambda(ctx context.Context, req any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	log := m.log.With(zap.Any("function", []string{m.FunctionName}))
	startTime := time.Now()

	resp, err := m.svc.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &m.FunctionName,
		Payload:      payload,
	})

	log = log.With(zap.Duration("duration", time.Since(startTime))).Named("exit")
	if err != nil {
		log.Error("", zap.Error(err))
		return nil, err
	}

	if resp.FunctionError != nil {
		err = fmt.Errorf("function error: %s: %w", *resp.FunctionError, errors.New(string(resp.Payload)))
		log.Error("", zap.Error(err))
		return nil, err
	}

	log.Info("")
	return resp.Payload, nil
}

func (m *LambdaMiddleware) eventFormat() string {
	if m.EventFormat == "" {
		return eventFormatHTTPJSON
	}
	return m.EventFormat
}

// Cleanup implements caddy.Cleanup.
// TODO: ensure all running processes are terminated.
func (m *LambdaMiddleware) Cleanup() error {
	return nil
}

func defaultReplyMeta() *ReplyMeta {
	return &ReplyMeta{
		Status:  http.StatusOK,
		Headers: map[string][]string{"content-type": {"application/json"}},
	}
}
