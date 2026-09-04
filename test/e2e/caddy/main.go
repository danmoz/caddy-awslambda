package main

import (
	"github.com/caddyserver/caddy/v2/cmd"
	_ "github.com/floj/caddy-awslambda"
)

func main() {
	caddycmd.Main()
}
