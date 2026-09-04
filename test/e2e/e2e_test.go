package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCaddyProcess(t *testing.T) {
	root := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "caddy")

	build := exec.Command("go", "build", "-o", binary, "./test/e2e/caddy")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build test Caddy: %v\n%s", err, output)
	}

	port := freePort(t)
	config := filepath.Join(t.TempDir(), "Caddyfile")
	configContents := fmt.Sprintf("http://127.0.0.1:%d {\n\trespond /health 200\n}\n", port)
	if err := os.WriteFile(config, []byte(configContents), 0o600); err != nil {
		t.Fatalf("write Caddyfile: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process := exec.CommandContext(ctx, binary, "run", "--config", config)
	process.Stdout = io.Discard
	process.Stderr = io.Discard
	if err := process.Start(); err != nil {
		t.Fatalf("start Caddy: %v", err)
	}
	defer func() {
		cancel()
		_ = process.Wait()
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("unexpected health status: %d", response.StatusCode)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("Caddy did not become ready at %s", url)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate test port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
