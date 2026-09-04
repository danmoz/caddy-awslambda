# caddy-awslambda

Caddy v2 module for dispatching requests to AWS Lambda.

This is a port of https://github.com/coopernurse/caddy-awslambda with less features but for Caddy 2.

## Installation

```
xcaddy build \
    --with github.com/floj/caddy-awslambda
```

## Usage

```
{
  order awslambda before file_server
}

http://localhost:8080 {
  log {
    output stderr
  }
  awslambda /services/* {
    function ForwardToSlack
    # endpoint http://127.0.0.1:3001
  }
}
```

The `endpoint` setting is optional. When omitted, the AWS SDK resolves the
standard Lambda endpoint. For local SAM testing, set it to the address used by
`sam local start-lambda`, for example `http://127.0.0.1:3001`.

## Event formats

The `event_format` setting selects both the request event and response format.

| Value            | Status    | Contract                                             |
| ---              | ---       | ---                                                  |
| `httpjson`       | Default   | `HTTPJSON-REQ` and `HTTPJSON-REP` envelopes.         |
| `api_gateway_v2` | Planned   | API Gateway HTTP API payload version 2.0 for Mangum. |
| `api_gateway_v1` | Planned   | API Gateway REST/proxy payload version 1.0.          |
| `alb`            | Planned   | Application Load Balancer Lambda event.              |
| `function_url`   | Planned   | Lambda Function URL event.                           |
| `lambda_at_edge` | Planned   | CloudFront Lambda@Edge event.                        |

### API Gateway v2

For API Gateway v2, request headers and duplicate query parameters use the
service's comma-joined representation, cookies use the `cookies` list, and the
raw path and query string are preserved in `rawPath` and `rawQueryString`.
Empty bodies are sent with base64 encoding disabled. Binary bodies use base64
encoding and set `isBase64Encoded` to `true`. Responses use `statusCode`,
`headers`, `cookies`, `body`, and `isBase64Encoded`; each response cookie is
emitted as a separate `Set-Cookie` header.

The initial API Gateway v2 adapter maps Caddy's request path directly to
`rawPath`; it does not trim a base path. AWS invocation errors, Lambda function
errors, timeouts, throttling, and malformed responses are returned as Caddy
handler errors and are not silently converted to successful responses.

## Development

You can lint and format code with mise commands:

```
mise format
mise lint
```

## License

Apache 2
