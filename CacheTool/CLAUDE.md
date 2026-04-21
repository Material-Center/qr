# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Release build (outputs logintool-release.apk)
./build.sh

# Or directly via Gradle
./gradlew assembleRelease   # Release APK → app/build/outputs/apk/release/app-release.apk
./gradlew assembleDebug     # Debug APK
./gradlew clean             # Clean build artifacts
./gradlew build             # All variants
```

No test runner is configured; the project ships with stub test files only.

## Project Purpose

Android APK for managing QQ/TIM login sessions on **rooted** Android devices. Extracts, encrypts/decrypts, and injects session credentials from `/data/data/com.tencent.mobileqq/` using root shell commands. Also exposes an HTTP API on port 9091 for remote automation.

## Architecture

The app follows a layered architecture:

```
MainActivity (UI)
    └── QQSessionService (Singleton facade)
            ├── FileManager       — root-based file I/O on QQ data dirs
            ├── SessionManager    — parses/transforms encrypted session data
            ├── DataImportService — imports JSON session into a fresh device
            └── QQFileGenerator   — creates QQ file structure from scratch
```

**HTTP Server** (`QQSessionHttpServer`, port 9091) runs as a singleton background thread, exposing the same operations via REST. See `README_HTTP_API.md` for the full endpoint spec.

**Result wrapper**: every service method returns `Result<T>` — always check `result.isSuccess()` before using the value.

**Root execution**: all privileged operations go through `CommandExecutor`, which shells out to `/system/bin/su`.

## Key Modules

| Module | Description |
|--------|-------------|
| `:app` | Main application |
| `:netbare-core` | VPN-based packet interception library |
| `:netbare-injector` | HTTP-layer interceptor built on netbare-core |

## Runtime Requirements

- Rooted Android device (minSdk 24)
- QQ (`com.tencent.mobileqq`) or TIM installed on device
- Internet + storage permissions granted at runtime

## Signing

Release signing config is in `sign.properties` (keystore password: `logintool`). `build.sh` uses this automatically.

## Notable Dependencies

- **BouncyCastle 1.70** — credential encryption/decryption
- **MMKV 2.2.3** — Tencent key-value storage (mirrors what QQ uses)
- **OkHttp 4.12.0** — remote server integration (`ServerApi`)
- **free_reflection** — access private Android APIs
