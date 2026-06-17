package main

import (
	"context"
	"runtime"
)

type App struct {
	ctx context.Context
}

type AppStatus struct {
	Name        string `json:"name"`
	Runtime     string `json:"runtime"`
	CoreReady   bool   `json:"coreReady"`
	Storage     string `json:"storage"`
	Description string `json:"description"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Status() AppStatus {
	return AppStatus{
		Name:        "Phone Task Client",
		Runtime:     runtime.GOOS + "/" + runtime.GOARCH,
		CoreReady:   true,
		Storage:     "SQLite",
		Description: "核心执行能力已可通过 CLI 使用；UI 壳已接入 Wails，后续页面将复用同一套核心能力。",
	}
}
