package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("miserver", flag.ContinueOnError)
	addr := fs.String("addr", ":9999", "listen address")
	seed := fs.String("seed", DefaultConfig().Seed, "AES key seed")
	iv := fs.String("iv", DefaultConfig().IV, "AES-CBC IV, 16 bytes")
	responseSeedPrefix := fs.String("response-seed-prefix", DefaultConfig().ResponseSeedPrefix, "response seed prefix")
	responseSkew := fs.Int("response-skew", DefaultConfig().ResponseSkew, "response decrypt skew in minutes")
	readHeaderTimeout := fs.Duration("read-header-timeout", 5*time.Second, "HTTP read header timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usage()
	}

	srv := NewServer(ServerConfig{
		Crypto: CryptoConfig{
			Seed:               *seed,
			IV:                 *iv,
			ResponseSeedPrefix: *responseSeedPrefix,
			ResponseSkew:       *responseSkew,
		},
	})
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: *readHeaderTimeout,
	}

	fmt.Fprintf(os.Stderr, "miserver listening on %s\n", *addr)
	return httpSrv.ListenAndServe()
}

func usage() error {
	return fmt.Errorf(`usage:
  miserver [flags]

examples:
  go run . -addr :9999
  go run ../miclient -base-url http://127.0.0.1:9999 shanghaitime`)
}
