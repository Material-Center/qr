package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	defaultEnvUpstreamURL = "http://210.16.170.132:1111/api"
	defaultEnvInternalKey = "cd5d1c1b4bd95fcb561d2a3f2b5407de82b9088d8b1f4eb3ebce9c28331ef42f"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("miserver", flag.ContinueOnError)
	bindIP := fs.String("bind-ip", "127.0.0.2", "local bind IP")
	authBindIP := fs.String("auth-bind-ip", "", "authorization bind IP; defaults to -bind-ip")
	uploadBindIP := fs.String("upload-bind-ip", "", "upload bind IP; defaults to -bind-ip")
	envBindIP := fs.String("env-bind-ip", "", "environment pool bind IP; defaults to -bind-ip")
	authPort := fs.Int("auth-port", 9999, "authorization API port")
	uploadPort := fs.Int("upload-port", 80, "upload API port")
	envPort := fs.Int("env-port", 8888, "environment pool API port")
	dbPath := fs.String("db", "miserver.db", "sqlite database path")
	seed := fs.String("seed", DefaultConfig().Seed, "AES key seed")
	iv := fs.String("iv", DefaultConfig().IV, "AES-CBC IV, 16 bytes")
	responseSeedPrefix := fs.String("response-seed-prefix", DefaultConfig().ResponseSeedPrefix, "response seed prefix")
	responseSkew := fs.Int("response-skew", DefaultConfig().ResponseSkew, "response decrypt skew in minutes")
	envUpstreamURL := fs.String("env-upstream-url", defaultEnvUpstreamURL, "main server base URL for environment APIs")
	envUpstreamPath := fs.String("env-upstream-path", "/internalTool/miEnv", "main server internal environment API path")
	envInternalKey := fs.String("env-internal-key", defaultEnvInternalKey, "fixed key for main server environment APIs")
	readHeaderTimeout := fs.Duration("read-header-timeout", 5*time.Second, "HTTP read header timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usage()
	}

	store, err := OpenStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	serverConfig := ServerConfig{
		Crypto: CryptoConfig{
			Seed:               *seed,
			IV:                 *iv,
			ResponseSeedPrefix: *responseSeedPrefix,
			ResponseSkew:       *responseSkew,
		},
		Store: store,
	}
	if *envUpstreamURL != "" {
		serverConfig.EnvProxy = &EnvProxyConfig{BaseURL: *envUpstreamURL, Path: *envUpstreamPath, Key: *envInternalKey}
	}
	srv := NewServer(serverConfig)
	addrs := buildServiceListenAddresses(
		fallbackBindIP(*authBindIP, *bindIP),
		fallbackBindIP(*uploadBindIP, *bindIP),
		fallbackBindIP(*envBindIP, *bindIP),
		*authPort, *uploadPort, *envPort,
	)
	servers := []namedHTTPServer{
		{name: "auth", server: &http.Server{Addr: addrs.Auth, Handler: srv.AuthHandler(), ReadHeaderTimeout: *readHeaderTimeout}},
		{name: "upload", server: &http.Server{Addr: addrs.Upload, Handler: srv.UploadHandler(), ReadHeaderTimeout: *readHeaderTimeout}},
		{name: "env", server: &http.Server{Addr: addrs.Env, Handler: srv.EnvHandler(), ReadHeaderTimeout: *readHeaderTimeout}},
	}
	return serveAll(servers)
}

type listenAddresses struct {
	Auth   string
	Upload string
	Env    string
}

type namedHTTPServer struct {
	name   string
	server *http.Server
}

func buildListenAddresses(bindIP string, authPort, uploadPort, envPort int) listenAddresses {
	return buildServiceListenAddresses(bindIP, bindIP, bindIP, authPort, uploadPort, envPort)
}

func buildServiceListenAddresses(authIP, uploadIP, envIP string, authPort, uploadPort, envPort int) listenAddresses {
	return listenAddresses{
		Auth:   authIP + ":" + strconv.Itoa(authPort),
		Upload: uploadIP + ":" + strconv.Itoa(uploadPort),
		Env:    envIP + ":" + strconv.Itoa(envPort),
	}
}

func fallbackBindIP(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func serveAll(servers []namedHTTPServer) error {
	listeners := make([]net.Listener, 0, len(servers))
	for _, item := range servers {
		listener, err := net.Listen("tcp", item.server.Addr)
		if err != nil {
			for _, opened := range listeners {
				_ = opened.Close()
			}
			return fmt.Errorf("%s listener %s: %w", item.name, item.server.Addr, err)
		}
		listeners = append(listeners, listener)
	}

	errCh := make(chan error, len(servers))
	var wg sync.WaitGroup
	for i, item := range servers {
		item := item
		listener := listeners[i]
		fmt.Fprintf(os.Stderr, "miserver %s listening on %s\n", item.name, item.server.Addr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := item.server.Serve(listener); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("%s listener %s: %w", item.name, item.server.Addr, err)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	if err, ok := <-errCh; ok {
		for _, item := range servers {
			_ = item.server.Close()
		}
		return err
	}
	return nil
}

func usage() error {
	return fmt.Errorf(`usage:
  miserver [flags]

examples:
  go run . -bind-ip 127.0.0.2 -db ./miserver.db
  go run ../miclient -base-url http://127.0.0.2:9999 shanghaitime
  go run ../miclient -base-url http://127.0.0.2:80 upload
  go run ../miclient -env-base-url http://127.0.0.2:8888 stats-env`)
}
