package caddyawslambda

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("awslambda", parseCaddyfile)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	m := &LambdaMiddleware{}
	err := m.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// UnmarshalCaddyfile configures the global directive from Caddyfile.
// Syntax:
//
//	awslambda [<matcher>] {
//	    function <function name>
//	    endpoint <url>
//	    region <region>
//	    access_key_id <access key id>
//	    secret_access_key <secret access key>
//	    session_token <session token>
//	    timeout  <duration>
//	}
func (m *LambdaMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}
		for d.NextBlock(0) {
			switch d.Val() {
			case "function":
				if m.FunctionName != "" {
					return d.Err("function already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.FunctionName = d.Val()
			case "endpoint":
				if m.Endpoint != "" {
					return d.Err("endpoint already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.Endpoint = d.Val()
			case "region":
				if m.Region != "" {
					return d.Err("region already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.Region = d.Val()
			case "access_key_id":
				if m.AccessKeyID != "" {
					return d.Err("access_key_id already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.AccessKeyID = d.Val()
			case "secret_access_key":
				if m.SecretAccessKey != "" {
					return d.Err("secret_access_key already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.SecretAccessKey = d.Val()
			case "session_token":
				if m.SessionToken != "" {
					return d.Err("session_token already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.SessionToken = d.Val()
			case "timeout":
				if m.Timeout != "" {
					return d.Err("timeout already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.Timeout = d.Val()
			case "event_format":
				if m.EventFormat != "" {
					return d.Err("event_format already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.EventFormat = d.Val()
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}
	return nil
}
