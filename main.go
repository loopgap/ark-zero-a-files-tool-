package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"arkkb/src/sidecar"
	"arkkb/src/utils/lock"
)

//go:embed docs/*.md
var docsFS embed.FS

func main() {
	setupSidecarLogging()
	log.Println("sidecar entry starting")

	pidLock, err := lock.NewPIDLock()
	if err == nil {
		_ = pidLock.Lock()
		defer pidLock.Unlock()
	}

	server, err := sidecar.New(readEmbeddedHelp)
	if err != nil {
		log.Fatalf("failed to start sidecar: %v", err)
	}
	defer server.Close()
	go func() {
		time.Sleep(12 * time.Second)
		server.RecoverWorkspace()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		server.Close()
		os.Exit(0)
	}()

	if err := server.ServeRPC(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func setupSidecarLogging() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(homeDir, ".arkkb", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}
	logPath := filepath.Join(logDir, "sidecar.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Printf("logging initialized: %s", logPath)
}

func readEmbeddedHelp(docID string) ([]byte, error) {
	docPath := filepath.Join("docs", "HELP.md")
	if docID == "developer" {
		docPath = filepath.Join("docs", "DEVELOPER.md")
	}
	data, err := docsFS.ReadFile(docPath)
	if err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("read help %s: %w", docID, err)
}
