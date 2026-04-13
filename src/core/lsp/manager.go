package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type LSPManager struct {
	instances map[string]*LSPInstance
	mu        sync.RWMutex
	onNotify  func(id string, method string, params interface{})
}

type LSPInstance struct {
	id         string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	ctx        context.Context
	cancel     context.CancelFunc
	reqID      int
	pendingReq map[int]chan interface{}
	reqMu      sync.Mutex
	onNotify   func(id string, method string, params interface{})
}

func NewLSPManager(onNotify func(id string, method string, params interface{})) *LSPManager {
	return &LSPManager{
		instances: make(map[string]*LSPInstance),
		onNotify:  onNotify,
	}
}

func (m *LSPManager) Start(id, executable string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.instances[id]; ok {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, executable, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	instance := &LSPInstance{
		id:         id,
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		ctx:        ctx,
		cancel:     cancel,
		pendingReq: make(map[int]chan interface{}),
		onNotify:   m.onNotify,
	}

	m.instances[id] = instance
	go instance.readLoop()

	return nil
}

func (m *LSPManager) Call(id string, method string, params interface{}) (interface{}, error) {
	m.mu.RLock()
	instance, ok := m.instances[id]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("LSP instance %s not found", id)
	}

	return instance.call(method, params)
}

func (m *LSPManager) Notify(id string, method string, params interface{}) error {
	m.mu.RLock()
	instance, ok := m.instances[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("LSP instance %s not found", id)
	}

	return instance.notify(method, params)
}

func (i *LSPInstance) call(method string, params interface{}) (interface{}, error) {
	i.reqMu.Lock()
	i.reqID++
	id := i.reqID
	ch := make(chan interface{}, 1)
	i.pendingReq[id] = ch
	i.reqMu.Unlock()

	// 确保在返回前删除待处理请求，防止通道泄漏
	defer func() {
		i.reqMu.Lock()
		delete(i.pendingReq, id)
		// 如果 channel 还没被读取，关闭它以释放资源
		i.reqMu.Unlock()
	}()

	err := i.send(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	// 使用 select 语句并添加超时，防止永久阻塞
	select {
	case res := <-ch:
		return res, nil
	case <-i.ctx.Done():
		return nil, i.ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("LSP call timeout after 30 seconds")
	}
}

func (i *LSPInstance) notify(method string, params interface{}) error {
	return i.send(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (i *LSPInstance) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(i.stdin, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

func (i *LSPInstance) readLoop() {
	reader := bufio.NewReader(i.stdout)
	tp := textproto.NewReader(reader)

	for {
		header, err := tp.ReadMIMEHeader()
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		lengthStr := header.Get("Content-Length")
		if lengthStr == "" {
			continue
		}

		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			continue
		}

		body := make([]byte, length)
		if _, err := io.ReadFull(reader, body); err != nil {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		if id, ok := msg["id"].(float64); ok {
			i.reqMu.Lock()
			if ch, ok := i.pendingReq[int(id)]; ok {
				ch <- msg["result"]
			}
			i.reqMu.Unlock()
		} else if method, ok := msg["method"].(string); ok {
			if i.onNotify != nil {
				i.onNotify(i.id, method, msg["params"])
			}
		}
	}
}

func (m *LSPManager) Stop(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, ok := m.instances[id]
	if !ok {
		return nil
	}

	if instance.cmd.Process != nil {
		_ = instance.cmd.Process.Signal(syscall.SIGTERM)
	}

	instance.cancel()
	err := instance.cmd.Wait()
	delete(m.instances, id)
	return err
}
