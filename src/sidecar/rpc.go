package sidecar

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"arkkb/src/bridge"
	"arkkb/src/core/config"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) ServeRPC(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var request rpcRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			if err := encoder.Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: err.Error()},
			}); err != nil {
				return err
			}
			continue
		}

		result, callErr := s.dispatch(request.Method, request.Params)
		response := rpcResponse{JSONRPC: "2.0", ID: request.ID}
		if callErr != nil {
			response.Error = &rpcError{Code: -32000, Message: callErr.Error()}
		} else {
			response.Result = result
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) dispatch(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "app.bootstrap":
		return map[string]string{"baseUrl": s.BaseURL()}, nil
	case "app.focusSync":
		return true, s.SyncOnFocus()
	case "workspace.get":
		return s.Bridge.GetWorkbenchState()
	case "workspace.open":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.OpenRecentWorkspace(payload.Path)
	case "workspace.addRoot":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.AddWorkspaceRoot(payload.Path)
	case "workspace.removeRoot":
		var payload struct {
			RootID string `json:"rootId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.RemoveWorkspaceRoot(payload.RootID)
	case "workspace.setActive":
		var payload struct {
			RootID string `json:"rootId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SetActiveRoot(payload.RootID)
	case "workspace.setDefault":
		var payload struct {
			RootID string `json:"rootId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SetDefaultRoot(payload.RootID)
	case "fs.createFile":
		var payload struct {
			ParentDir string `json:"parentDir"`
			Name      string `json:"name"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.CreateFile(payload.ParentDir, payload.Name)
	case "fs.createFolder":
		var payload struct {
			ParentDir string `json:"parentDir"`
			Name      string `json:"name"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.CreateFolder(payload.ParentDir, payload.Name)
	case "fs.rename":
		var payload struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.Rename(payload.Path, payload.Name)
	case "fs.delete":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SoftDelete(payload.Path)
	case "fs.read":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.ReadFile(payload.Path)
	case "fs.listDir":
		var payload struct {
			Path   string `json:"path"`
			RootID string `json:"rootId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.ListDirectoryChildren(payload.RootID, payload.Path)
	case "fs.save":
		var payload struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SaveFile(payload.Path, payload.Content)
	case "fs.saveBinary":
		var payload struct {
			Path          string `json:"path"`
			ContentBase64 string `json:"contentBase64"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SaveBinaryFile(payload.Path, payload.ContentBase64)
	case "fs.openExternal":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.OpenWithExternalApp(payload.Path)
	case "archive.list":
		var payload struct {
			FolderID string `json:"folderId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.ListVirtualFolderItems(payload.FolderID)
	case "archive.browse":
		var payload bridge.ArchiveBrowseRequest
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.BrowseArchive(payload)
	case "archive.create":
		var payload struct {
			Name string `json:"name"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.CreateVirtualFolder(payload.Name)
	case "archive.rename":
		var payload struct {
			FolderID string `json:"folderId"`
			Name     string `json:"name"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.RenameVirtualFolder(payload.FolderID, payload.Name)
	case "archive.delete":
		var payload struct {
			FolderID string `json:"folderId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.DeleteVirtualFolder(payload.FolderID)
	case "archive.attach":
		var payload struct {
			Path     string `json:"path"`
			FolderID string `json:"folderId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.AttachFileToVirtualFolder(payload.Path, payload.FolderID)
	case "archive.detach":
		var payload struct {
			Path     string `json:"path"`
			FolderID string `json:"folderId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.DetachFileFromVirtualFolder(payload.Path, payload.FolderID)
	case "archive.createFile":
		var payload struct {
			FolderID        string `json:"folderId"`
			Name            string `json:"name"`
			ParentPath      string `json:"parentPath"`
			PreferredRootID string `json:"preferredRootId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.CreateVirtualFile(payload.FolderID, payload.Name, payload.ParentPath, payload.PreferredRootID)
	case "search.query":
		var payload bridge.SearchOptions
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		results, err := s.Bridge.SearchFiles(payload)
		if err != nil {
			return nil, err
		}
		if payload.AutoCategory == "" {
			return results, nil
		}
		filtered := make([]bridge.SearchHit, 0, len(results))
		for _, item := range results {
			if strings.EqualFold(item.Extension, payload.AutoCategory) {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	case "settings.get":
		return s.Bridge.GetConfig()
	case "settings.save":
		var payload struct {
			Config config.AppConfig `json:"config"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.SaveConfig(&payload.Config)
	case "help.list":
		state, err := s.Bridge.GetWorkbenchState()
		if err != nil {
			return nil, err
		}
		return state.HelpDocs, nil
	case "help.read":
		var payload struct {
			DocID string `json:"docId"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return s.Bridge.ReadHelpDoc(payload.DocID)
	case "recent.record":
		var payload struct {
			Path string `json:"path"`
		}
		if err := decodeParams(params, &payload); err != nil {
			return nil, err
		}
		return true, s.Bridge.RecordRecentItem(payload.Path)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func decodeParams(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func ShutdownContext() context.Context {
	return context.Background()
}
