package bridge

import (
	"arkkb/src/core/lsp"
	"arkkb/src/core/storage"
	"context"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Bridge struct {
	app     *application.App
	lspMgr  *lsp.LSPManager
	storage *storage.StorageManager
}

func NewBridge(storage *storage.StorageManager) *Bridge {
	b := &Bridge{
		storage: storage,
	}
	// Note: In a real production environment, we would bridge this to Wails events.
	b.lspMgr = lsp.NewLSPManager(b.onLSPNotify)
	return b
}

func (b *Bridge) SetApp(app *application.App) {
	b.app = app
}

func (b *Bridge) onLSPNotify(id string, method string, params interface{}) {
	log.Printf("[LSP Notify] ID: %s, Method: %s", id, method)
	// Placeholder for Wails v3 Event Emit:
	// If the API stabilizes, use: application.Get().EmitEvent(...)
}

func (b *Bridge) StartLSP(id string, executable string, args []string) error {
	return b.lspMgr.Start(id, executable, args)
}

func (b *Bridge) StopLSP(id string) error {
	return b.lspMgr.Stop(id)
}

func (b *Bridge) LSPCall(id string, method string, params interface{}) (interface{}, error) {
	return b.lspMgr.Call(id, method, params)
}

func (b *Bridge) LSPNotify(id string, method string, params interface{}) error {
	return b.lspMgr.Notify(id, method, params)
}

func (b *Bridge) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (b *Bridge) SaveFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (b *Bridge) OnStartup(ctx context.Context) {
}
