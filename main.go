package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"arkkb/src/bridge"
	"arkkb/src/core/storage"
	"arkkb/src/core/sync"
	"arkkb/src/utils/lock"

	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/dist/*
var assets embed.FS

func main() {
	// 1. PID Lock (Chapter 9.3)
	pidLock, err := lock.NewPIDLock()
	if err != nil {
		log.Fatalf("Failed to initialize PID lock: %v", err)
	}
	if err := pidLock.Lock(); err != nil {
		log.Fatalf("Application is already running: %v", err)
	}
	defer pidLock.Unlock()

	// 2. Storage Manager (Chapter 7)
	sm := storage.NewStorageManager()
	if err := sm.Init(); err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer sm.Close()

	// 3. Sync Engine (Chapter 3.1)
	syncEngine := sync.NewSyncEngine(sm)

	// 4. Bridge (Chapter 6)
	br := bridge.NewBridge(sm)

	// 5. Wails Application
	app := application.New(application.Options{
		Name:        "ArkKB",
		Description: "Absolute Lightweight Knowledge Base",
		Services: []application.Service{
			application.NewService(br),
		},
		Assets: application.AssetOptions{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Chapter 4.1: AssetServer Hijacking for Native Streaming
				if strings.HasPrefix(r.URL.Path, "/file/") {
					filePath := strings.TrimPrefix(r.URL.Path, "/file/")
					http.ServeFile(w, r, filePath)
					return
				}
				application.AssetFileServerFS(assets).ServeHTTP(w, r)
			}),
		},
	})

	br.SetApp(app)

	// Focus-Driven Sync (Chapter 3.1)
	app.On(events.Common.WindowFocus, func(e *application.WailsEvent) {
		log.Println("[Focus-Driven Sync] 窗口获得焦点：触发目录树 mtime 惰性比对...")
		wd, _ := os.Getwd()
		if err := syncEngine.OnFocus(wd); err != nil {
			if err == sync.ErrTooManyChanges {
				log.Println("[Sync] 检测到大量文件变更，已发出 UI 提示事件")
				app.EmitEvent("sync:too_many_changes", nil)
			} else {
				log.Printf("[Sync] 同步错误: %v", err)
			}
		}
	})

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  "ArkKB - Minimalist Knowledge Base",
		Width:  1024,
		Height: 768,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 30,
			TitleBar:                application.MacTitleBarHiddenInset,
			Appearance:              application.NSAppearanceNameDarkAqua,
			WebviewIsTransparent:    true,
			WindowIsTransparent:     true,
		},
	})

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
