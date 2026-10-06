package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	return runWithOutput(args, os.Stdout)
}

func runWithOutput(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("miclient", flag.ContinueOnError)
	baseURL := fs.String("base-url", defaultBaseURL, "activation server base URL")
	uploadBaseURL := fs.String("upload-base-url", defaultUploadBaseURL, "upload server base URL")
	envBaseURL := fs.String("env-base-url", defaultEnvBaseURL, "environment pool server base URL")
	seed := fs.String("seed", DefaultConfig().Seed, "AES key seed")
	iv := fs.String("iv", DefaultConfig().IV, "AES-CBC IV, 16 bytes")
	envSeed := fs.String("env-seed", DefaultEnvConfig().Seed, "environment pool AES key seed")
	envIV := fs.String("env-iv", DefaultEnvConfig().IV, "environment pool AES-CBC IV, 16 bytes")
	device := fs.String("device", "", "device ID")
	deviceCode := fs.String("device-code", "", "device code/model for environment pool APIs")
	envType := fs.String("env-type", "QQ888", "environment type")
	serialBackupName := fs.String("serial-backup-name", "", "serial backup package name")
	androidID := fs.String("android-id", "", "Android ID")
	envKey := fs.String("key", "", "environment userkey/secret")
	envID := fs.Int("env-id", 0, "environment ID")
	maxUsage := fs.Int("max-usage", -1, "maximum usage filter; -1 omits it")
	minUsage := fs.Int("min-usage", -1, "minimum usage filter; -1 omits it")
	minMade := fs.Int("min-made-count", -1, "minimum made count; -1 omits it")
	maxMade := fs.Int("max-made-count", -1, "maximum made count; -1 omits it")
	olderThanDays := fs.Int("older-than-days", -1, "older-than-days filter; -1 omits it")
	minDays := fs.Int("min-days", -1, "minimum age window in days; -1 omits it")
	maxDays := fs.Int("max-days", -1, "maximum age window in days; -1 omits it")
	cooldownDays := fs.Int("cooldown-days", -1, "make cooldown in days; -1 omits it")
	sortMode := fs.String("sort", "", "environment sort mode")
	limit := fs.Int("limit", -1, "query limit; -1 omits it")
	offset := fs.Int("offset", -1, "query offset; -1 omits it")
	frozen := fs.Int("frozen", -1, "frozen filter: 0/1; -1 omits it")
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
	uploadClient := NewClient(*uploadBaseURL, CryptoConfig{Seed: *seed, IV: *iv})
	uploadClient.SetTimeout(*timeout)
	envClient := NewEnvClient(*envBaseURL, CryptoConfig{Seed: *envSeed, IV: *envIV})
	envClient.SetTimeout(*timeout)

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
		resp, err = uploadClient.Upload(*device, *currentTime, *phone, *account, *password)
	case "add-env":
		if *device == "" || *deviceCode == "" || *serialBackupName == "" || *androidID == "" || *envKey == "" {
			return fmt.Errorf("-device, -device-code, -serial-backup-name, -android-id and -key are required")
		}
		resp, err = envClient.AddEnv(EnvRecord{
			DeviceCode:       *deviceCode,
			DeviceID:         *device,
			Type:             *envType,
			SerialBackupName: *serialBackupName,
			AndroidID:        *androidID,
			Key:              *envKey,
		})
	case "get-env":
		resp, err = envClient.GetEnv(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "get-env-enhanced":
		resp, err = envClient.GetEnvEnhanced(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "get-env-enhanced2":
		resp, err = envClient.GetEnvEnhanced2(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "get-env-for-make":
		resp, err = envClient.GetEnvForMake(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "make-success":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		resp, err = envClient.MakeSuccess(*envID)
	case "increase-make-count", "decrease-make-count", "reset-make-count":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		switch fs.Arg(0) {
		case "increase-make-count":
			resp, err = envClient.IncreaseMakeCount(*envID)
		case "decrease-make-count":
			resp, err = envClient.DecreaseMakeCount(*envID)
		default:
			resp, err = envClient.ResetMakeCount(*envID)
		}
	case "get-env-fixed":
		if *device == "" {
			return fmt.Errorf("-device is required")
		}
		resp, err = envClient.GetEnvFixed(*device)
	case "stats-by-type":
		resp, err = envClient.StatsByType()
	case "stats-make-progress":
		resp, err = envClient.StatsMakeProgress()
	case "total-env":
		resp, err = envClient.Total()
	case "available-env":
		resp, err = envClient.Available()
	case "frozen-env":
		resp, err = envClient.Frozen()
	case "unused-env":
		resp, err = envClient.Unused()
	case "query-env-list":
		resp, err = envClient.QueryEnvList(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "query-env":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		resp, err = envClient.QueryEnv(*envID)
	case "freeze-env":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		resp, err = envClient.FreezeEnv(*envID)
	case "unfreeze-env":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		resp, err = envClient.UnfreezeEnv(*envID)
	case "freeze-by-condition":
		resp, err = envClient.FreezeByCondition(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "unfreeze-by-condition":
		resp, err = envClient.UnfreezeByCondition(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "delete-env":
		if *envID <= 0 {
			return fmt.Errorf("-env-id is required")
		}
		resp, err = envClient.DeleteEnv(*envID)
	case "delete-by-condition":
		resp, err = envClient.DeleteByCondition(envFilterFromFlags(*envType, *deviceCode, *device, *serialBackupName, *androidID, *envKey, *frozen, *limit, *offset, *maxUsage, *minUsage, *olderThanDays, *minDays, *maxDays, *minMade, *maxMade, *cooldownDays, *sortMode))
	case "clean-env":
		if *olderThanDays >= 0 {
			resp, err = envClient.CleanEnvOlderThan(*olderThanDays)
		} else {
			resp, err = envClient.CleanEnv()
		}
	case "query-by-device":
		if *device == "" {
			return fmt.Errorf("-device is required")
		}
		resp, err = envClient.QueryByDevice(*device, *limit)
	case "stats-env":
		resp, err = envClient.Stats()
	default:
		return usage()
	}
	if err != nil {
		return err
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func envFilterFromFlags(envType, deviceCode, device, serialBackupName, androidID, envKey string, frozen, limit, offset, maxUsage, minUsage, olderThanDays, minDays, maxDays, minMade, maxMade, cooldownDays int, sort string) EnvFilter {
	return EnvFilter{
		Type:             envType,
		DeviceCode:       deviceCode,
		DeviceID:         device,
		SerialBackupName: serialBackupName,
		AndroidID:        androidID,
		Key:              envKey,
		Frozen:           intPtrIfSet(frozen),
		Limit:            intPtrIfSet(limit),
		Offset:           intPtrIfSet(offset),
		MaxUsage:         intPtrIfSet(maxUsage),
		MinUsage:         intPtrIfSet(minUsage),
		MinMadeCount:     intPtrIfSet(minMade),
		MaxMadeCount:     intPtrIfSet(maxMade),
		OlderThanDays:    intPtrIfSet(olderThanDays),
		MinDays:          intPtrIfSet(minDays),
		MaxDays:          intPtrIfSet(maxDays),
		CooldownDays:     intPtrIfSet(cooldownDays),
		Sort:             sort,
	}
}

func intPtrIfSet(value int) *int {
	if value < 0 {
		return nil
	}
	return &value
}

func usage() error {
	return fmt.Errorf(`usage:
  miclient [flags] shanghaitime
  miclient [flags] -device <id> get-device
  miclient [flags] -device <id> -code <code> use-code
  miclient [flags] -device <id> -current-time <time> -phone <phone> -account <account> -password <password> upload

environment pool:
  miclient [flags] -device <id> -device-code <model> [-env-type QQ888] -serial-backup-name <name> -android-id <id> -key <key> add-env
  miclient [flags] [-env-type QQ888] [-device-code cepheus] [-device <id>] [-max-usage 1] [-older-than-days 3] get-env
  miclient [flags] get-env-enhanced | get-env-enhanced2 | get-env-for-make
  miclient [flags] -env-id <id> make-success
  miclient [flags] -device <id> get-env-fixed
  miclient [flags] [filters] query-env-list
  miclient [flags] -env-id <id> query-env
  miclient [flags] -env-id <id> freeze-env
  miclient [flags] -env-id <id> unfreeze-env
  miclient [flags] -env-id <id> delete-env
  miclient [flags] -device <id> [-limit n] query-by-device
  miclient [flags] clean-env
  miclient [flags] stats-env`)
}
