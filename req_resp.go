package caddyawslambda

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	eventFormatHTTPJSON     = "httpjson"
	eventFormatAPIGatewayV2 = "api_gateway_v2"
)

// parseReply unpacks the Lambda response data into a Reply.
// If the reply is a JSON object with a 'type' field equal to 'HTTPJSON-REP', then
// data will be unmarshaled directly as a Reply struct.
//
// If data is not a JSON object, or the object's type field is omitted or set to
// a string other than 'HTTPJSON-REP', then data will be set as the Reply.body
// and Reply.meta will contain a default struct with a 200 status and
// a content-type header of 'application/json'.
func parseReply(data []byte) (*Reply, error) {
	if len(data) > 0 && data[0] == '{' {
		var rep Reply
		err := json.Unmarshal(data, &rep)
		if err == nil && rep.Type == "HTTPJSON-REP" {
			if rep.Meta == nil {
				rep.Meta = defaultReplyMeta()
			}
			return &rep, nil
		}
	}

	return &Reply{
		Type: "HTTPJSON-REP",
		Meta: defaultReplyMeta(),
		Body: string(data),
	}, nil
}

func newRequestForFormat(r *http.Request, format string) (any, error) {
	body, err := readRequestBody(r)
	if err != nil {
		return nil, err
	}

	switch format {
	case eventFormatHTTPJSON:
		return newHTTPJSONRequest(r, body), nil
	case eventFormatAPIGatewayV2:
		return newAPIGatewayV2Request(r, body), nil
	default:
		return nil, fmt.Errorf("unsupported event format %q", format)
	}
}

func readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func newHTTPJSONRequest(r *http.Request, body []byte) *Request {
	return &Request{
		Type: "HTTPJSON-REQ",
		Meta: newRequestMeta(r),
		Body: string(body),
	}
}

func newAPIGatewayV2Request(r *http.Request, body []byte) *APIGatewayV2Request {
	return &APIGatewayV2Request{
		Version:               "2.0",
		RouteKey:              "$default",
		RawPath:               r.URL.EscapedPath(),
		RawQueryString:        r.URL.RawQuery,
		Headers:               apiGatewayV2Headers(r),
		Cookies:               apiGatewayV2Cookies(r),
		QueryStringParameters: apiGatewayV2QueryParameters(r),
		RequestContext: APIGatewayV2RequestContext{
			HTTP: APIGatewayV2HTTPContext{
				Method:    r.Method,
				Path:      r.URL.Path,
				Protocol:  r.Proto,
				SourceIP:  requestSourceIP(r),
				UserAgent: r.UserAgent(),
			},
		},
		Body:            apiGatewayV2Body(r, body),
		IsBase64Encoded: apiGatewayV2IsBase64Encoded(r, body),
	}
}

func apiGatewayV2Headers(r *http.Request) map[string]string {
	headers := make(map[string]string, len(r.Header)+1)
	for key, values := range r.Header {
		key = strings.ToLower(key)
		if key == "cookie" {
			continue
		}
		headers[key] = strings.Join(values, ",")
	}
	if _, ok := headers["host"]; !ok {
		headers["host"] = r.Host
	}
	return headers
}

func apiGatewayV2Cookies(r *http.Request) []string {
	cookies := r.Cookies()
	values := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		values = append(values, cookie.String())
	}
	return values
}

func apiGatewayV2QueryParameters(r *http.Request) map[string]string {
	query := r.URL.Query()
	parameters := make(map[string]string, len(query))
	for key, values := range query {
		parameters[key] = strings.Join(values, ",")
	}
	return parameters
}

func requestSourceIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func apiGatewayV2Body(r *http.Request, body []byte) string {
	if apiGatewayV2IsBase64Encoded(r, body) {
		return base64.StdEncoding.EncodeToString(body)
	}
	return string(body)
}

func apiGatewayV2IsBase64Encoded(r *http.Request, body []byte) bool {
	if len(body) == 0 {
		return false
	}
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/javascript") || strings.HasPrefix(contentType, "application/xml") ||
		strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return false
	}
	return !utf8.Valid(body) || contentType != ""
}

// APIGatewayV2Request is the HTTP API payload format consumed by Mangum.
type APIGatewayV2Request struct {
	Version               string                     `json:"version"`
	RouteKey              string                     `json:"routeKey"`
	RawPath               string                     `json:"rawPath"`
	RawQueryString        string                     `json:"rawQueryString"`
	Headers               map[string]string          `json:"headers"`
	Cookies               []string                   `json:"cookies,omitempty"`
	QueryStringParameters map[string]string          `json:"queryStringParameters,omitempty"`
	RequestContext        APIGatewayV2RequestContext `json:"requestContext"`
	Body                  string                     `json:"body,omitempty"`
	IsBase64Encoded       bool                       `json:"isBase64Encoded"`
}

type APIGatewayV2RequestContext struct {
	HTTP APIGatewayV2HTTPContext `json:"http"`
}

type APIGatewayV2HTTPContext struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Protocol  string `json:"protocol"`
	SourceIP  string `json:"sourceIp"`
	UserAgent string `json:"userAgent"`
}

// newRequestMeta returns a new RequestMeta based on the HTTP request
func newRequestMeta(r *http.Request) *RequestMeta {
	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[strings.ToLower(k)] = v
	}
	return &RequestMeta{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Host:    r.Host,
		Proto:   r.Proto,
		Headers: headers,
	}
}

// Request represents a single HTTP request.  It will be serialized as JSON
// and sent to the AWS Lambda function as the function payload.
type Request struct {
	// Set to the constant "HTTPJSON-REQ"
	Type string `json:"type"`
	// Metadata about the HTTP request
	Meta *RequestMeta `json:"meta"`
	// HTTP request body (may be empty)
	Body string `json:"body"`
}

// RequestMeta represents HTTP metadata present on the request
type RequestMeta struct {
	// HTTP method used by client (e.g. GET or POST)
	Method string `json:"method"`

	// Path portion of URL without the query string
	Path string `json:"path"`

	// Query string (without '?')
	Query string `json:"query"`

	// Host field from net/http Request, which may be of the form host:port
	Host string `json:"host"`

	// Proto field from net/http Request, for example "HTTP/1.1"
	Proto string `json:"proto"`

	// HTTP request headers
	Headers map[string][]string `json:"headers"`
}

// Reply encapsulates the response from a Lambda invocation.
// AWS Lambda functions should return a JSON object that matches this format.
type Reply struct {
	// Must be set to the constant "HTTPJSON-REP"
	Type string `json:"type"`
	// Reply metadata. If omitted, a default 200 status with empty headers will be used.
	Meta *ReplyMeta `json:"meta"`
	// Response body
	Body string `json:"body"`
	// Encoding of Body - Valid values: "", "base64"
	BodyEncoding string `json:"bodyEncoding"`
}

// ReplyMeta encapsulates HTTP response metadata that the lambda function wishes
// Caddy to set on the HTTP response.
//
// *NOTE* that header values must be encoded as string arrays
type ReplyMeta struct {
	// HTTP status code (e.g. 200 or 404)
	Status int `json:"status"`
	// HTTP response headers
	Headers map[string][]string `json:"headers"`
}
