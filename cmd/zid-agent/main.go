package main

import (
	"context"
	"flag"
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("zid-agent version %s (built %s)\n", Version, BuildTime)
		return
	}

	// Create log manager with capacity for 500 messages
	logMgr := agentui.NewLogManager(500)
	logMgr.Addf("ZID Agent v%s starting...", Version)

	// Setup config manager (persisted in ~/.zid-agent/config.json)
	cfg := agentui.DefaultConfig()
	cfgPath, err := agentui.DefaultConfigPath()
	if err != nil {
		logMgr.Addf("Warning: could not resolve config path: %v (using defaults)", err)
	}
	cfgMgr := agentui.NewConfigManager(cfgPath, cfg)
	if cfgPath != "" {
		if err := cfgMgr.LoadFromDisk(); err != nil {
			logMgr.Addf("Warning: could not load config: %v (using defaults)", err)
		}
	}

	// Persist logs to ~/.zid-agent/logs.txt with rotation (max 1MB)
	if logPath, err := agentui.DefaultLogPath(); err == nil {
		logMgr.AddSink(agentui.NewFileLogSink(logPath, agentui.DefaultMaxLogBytes))
		logMgr.Addf("Log file: %s", logPath)
	} else {
		logMgr.Addf("Warning: could not resolve log path: %v (logs only in memory)", err)
	}

	statusMgr := agentui.NewStatusManager()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Fyne app on main thread
	fyneApp := app.New()

	fyneApp.Lifecycle().SetOnStopped(func() {
		logMgr.Add("Application stopping...")
		cancel()
	})

	// Setup system tray if supported
	if deskApp, ok := fyneApp.(desktop.App); ok {
		setupSystemTray(deskApp, fyneApp, logMgr, statusMgr, cfgMgr, cancel, Version, BuildTime)
	} else {
		logMgr.Add("Warning: System tray not supported on this platform/driver")
		showLogsWindow(fyneApp, logMgr)
	}

	// Start heartbeat goroutine
	go runHeartbeat(ctx, logMgr, statusMgr, cfgMgr, Version)

	// Run Fyne event loop (blocks until quit)
	fyneApp.Run()
}
