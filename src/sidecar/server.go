package sidecar

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"arkkb/src/bridge"
	"arkkb/src/core/file"
	"arkkb/src/core/render"
	"arkkb/src/core/storage"
	"arkkb/src/core/sync"
)

type Server struct {
	Bridge     *bridge.Bridge
	storage    *storage.StorageManager
	syncEngine *sync.SyncEngine
	httpServer *http.Server
	baseURL    string
	readHelp   func(string) ([]byte, error)
}

func New(readHelp func(string) ([]byte, error)) (*Server, error) {
	if homeDir, err := os.UserHomeDir(); err == nil {
		log.Printf("initializing storage in %s", filepath.Join(homeDir, ".arkkb"))
	}
	storageManager := storage.NewStorageManager()
	if err := storageManager.Init(); err != nil {
		return nil, err
	}

	syncEngine := sync.NewSyncEngine(storageManager)
	bridgeSvc := bridge.NewBridge(storageManager, syncEngine, readHelp)
	server := &Server{
		Bridge:     bridgeSvc,
		storage:    storageManager,
		syncEngine: syncEngine,
		readHelp:   readHelp,
	}
	if err := server.startHTTP(); err != nil {
		storageManager.Close()
		return nil, err
	}
	log.Printf("sidecar HTTP listening on %s", server.baseURL)

	cfg, err := storageManager.KV.GetAppConfig()
	if err == nil {
		log.Printf("startup config loaded: roots=%d virtualFolders=%d", len(cfg.Workspace.Roots), len(cfg.VirtualFolders))
		go func(lastWorkspace string) {
			time.Sleep(6 * time.Second)
			_ = syncEngine.OnFocus(lastWorkspace)
		}(cfg.LastWorkspace)
	}
	return server, nil
}

func (s *Server) Close() {
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
	if s.storage != nil {
		s.storage.Close()
	}
}

func (s *Server) BaseURL() string {
	return s.baseURL
}

func (s *Server) SyncOnFocus() error {
	cfg, err := s.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	go func(lastWorkspace string) {
		_ = s.syncEngine.OnFocus(lastWorkspace)
	}(cfg.LastWorkspace)
	return nil
}

func (s *Server) startHTTP() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.baseURL = "http://" + listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("/file/", func(w http.ResponseWriter, r *http.Request) {
		filePath, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/file/"))
		if err != nil {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		if strings.Contains(filePath, "..") {
			http.Error(w, "forbidden path traversal", http.StatusForbidden)
			return
		}
		normalizedPath, err := s.Bridge.NormalizeWorkspacePath(filePath)
		if err != nil {
			http.Error(w, "file not accessible", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, filepath.FromSlash(normalizedPath))
	})
	mux.HandleFunc("/render/", func(w http.ResponseWriter, r *http.Request) {
		filePath, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/render/"))
		if err != nil {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		if strings.Contains(filePath, "..") {
			http.Error(w, "forbidden path traversal", http.StatusForbidden)
			return
		}
		normalizedPath, err := s.Bridge.NormalizeWorkspacePath(filePath)
		if err != nil {
			http.Error(w, "file not accessible", http.StatusForbidden)
			return
		}
		htmlBytes, err := render.RenderMarkdown(filepath.FromSlash(normalizedPath))
		if err != nil {
			http.Error(w, "render failed", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlBytes)
	})
	mux.HandleFunc("/help/", func(w http.ResponseWriter, r *http.Request) {
		docID := strings.TrimPrefix(r.URL.Path, "/help/")
		if s.readHelp == nil {
			http.Error(w, "help docs are unavailable", http.StatusInternalServerError)
			return
		}
		docPath := filepath.Join("docs", "HELP.md")
		if docID == "developer" {
			docPath = filepath.Join("docs", "DEVELOPER.md")
		}
		docBytes, err := s.readHelp(docID)
		if err != nil {
			http.Error(w, fmt.Sprintf("help doc not found: %s", docID), http.StatusNotFound)
			return
		}
		htmlBytes, err := render.RenderMarkdownBytes(docPath, docBytes)
		if err != nil {
			http.Error(w, "render failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlBytes)
	})

	s.httpServer = &http.Server{Handler: mux}
	go func() {
		_ = s.httpServer.Serve(listener)
	}()
	return nil
}

func (s *Server) RecoverWorkspace() {
	cfg, err := s.storage.KV.GetAppConfig()
	if err != nil {
		return
	}
	log.Printf("recover workspace: roots=%d lastWorkspace=%s", len(cfg.Workspace.Roots), cfg.LastWorkspace)
	if len(cfg.Workspace.Roots) == 0 && cfg.LastWorkspace != "" && cfg.LastWorkspace != "." {
		_ = file.GlobalRescue(cfg.LastWorkspace)
	}
	for _, root := range cfg.Workspace.Roots {
		_ = file.GlobalRescue(filepath.FromSlash(root.Path))
	}
}
