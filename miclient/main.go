package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	fs := flag.NewFlagSet("miclient", flag.ContinueOnError)
	baseURL := fs.String("base-url", defaultBaseURL, "activation server base URL")
	seed := fs.String("seed", DefaultConfig().Seed, "AES key seed")
	iv := fs.String("iv", DefaultConfig().IV, "AES-CBC IV, 16 bytes")
	device := fs.String("device", "", "device ID")
	code := fs.String("code", "", "activation code")
	currentTime := fs.String("current-time", "", "current time sent to /上传")
	phone := fs.String("phone", "", "phone sent to /上传")
	account := fs.String("account", "", "account sent to /上传")
	password := fs.String("password", "", "password sent to /上传")
	timeout := fs.Duration("timeout", 10*time.Second, "HTTP timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return usage()
	}

	client := NewClient(*baseURL, CryptoConfig{Seed: *seed, IV: *iv})
	client.SetTimeout(*timeout)

	var (
		resp APIResponse
		err  error
	)

	switch fs.Arg(0) {
	case "shanghaitime":
		resp, err = client.ShanghaiTime()
	case "get-device":
		if *device == "" {
			return fmt.Errorf("-device is required")
		}
		resp, err = client.GetDevice(*device)
	case "use-code":
		if *device == "" || *code == "" {
			return fmt.Errorf("-device and -code are required")
		}
		resp, err = client.UseCode(*device, *code)
	case "upload":
		if *device == "" || *currentTime == "" || *phone == "" || *account == "" || *password == "" {
			return fmt.Errorf("-device, -current-time, -phone, -account and -password are required")
		}
		resp, err = client.Upload(*device, *currentTime, *phone, *account, *password)
	default:
		return usage()
	}
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func usage() error {
	return fmt.Errorf(`usage:
  miclient [flags] shanghaitime
  miclient [flags] -device <id> get-device
  miclient [flags] -device <id> -code <code> use-code
  miclient [flags] -device <id> -current-time <time> -phone <phone> -account <account> -password <password> upload`)
}
