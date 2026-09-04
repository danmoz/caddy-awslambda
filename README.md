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

## License

Apache 2
