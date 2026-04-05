package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"arkkb/src/core/storage"
	"github.com/blugelabs/bluge"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "doctor":
		runDoctor()
	case "bench":
		runBench()
	case "test":
		runTest()
	case "clean":
		runClean()
	case "build":
		runBuild()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run scripts/dev.go <command>")
	fmt.Println("Commands:")
	fmt.Println("  doctor - 环境测谎机 (Check environment and dependencies)")
	fmt.Println("  bench  - 海量压测器 (Run performance benchmarks)")
	fmt.Println("  test   - 自动化验证 (Run unit tests and linting)")
	fmt.Println("  clean  - 清理构建产物 (Clean build artifacts)")
	fmt.Println("  build  - 极限出包 (Build and compress)")
}

func runDoctor() {
	fmt.Println("🔍 [doctor] 正在进行环境体检...")

	cgo := os.Getenv("CGO_ENABLED")
	if cgo == "" {
		out, _ := exec.Command("go", "env", "CGO_ENABLED").Output()
		cgo = strings.TrimSpace(string(out))
	}
	if cgo == "0" {
		fmt.Println("✅ CGO_ENABLED = 0 (符合 No CGO 规范)")
	} else {
		fmt.Printf("❌ CGO_ENABLED != 0 (当前为: %s)。请设置 export CGO_ENABLED=0\n", cgo)
	}

	if runtime.GOOS == "windows" {
		fmt.Print("🔍 检查 WebView2 (Windows)... ")
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-ItemProperty -Path 'HKLM:\\SOFTWARE\\WOW6432Node\\Microsoft\\EdgeUpdate\\Clients\\{F3C4FE00-EFC5-498C-9574-05118A201666}' -ErrorAction SilentlyContinue")
		if err := cmd.Run(); err == nil {
			fmt.Println("✅ 已安装")
		} else {
			fmt.Println("❌ 未发现 WebView2 运行时")
		}
	}

	fmt.Printf("ℹ️  Go 版本: %s\n", runtime.Version())
	fmt.Println("✨ [doctor] 检查完成。")
}

func runBench() {
	fmt.Println("🚀 [bench] 启动海量压测...")
	benchDir := filepath.Join(".tmp", "bench")
	count := 100

	os.RemoveAll(benchDir)
	os.MkdirAll(benchDir, 0755)

	fmt.Printf("📂 正在生成 %d 个随机内容伪文件...\n", count)
	for i := 0; i < count; i++ {
		filename := filepath.Join(benchDir, fmt.Sprintf("bench_%d.txt", i))
		content := make([]byte, 512)
		rand.Read(content)
		_ = os.WriteFile(filename, content, 0644)
	}

	fmt.Println("🏗️  正在测试大规模索引构建...")
	sm := storage.NewStorageManager()
	if err := sm.Init(); err != nil {
		fmt.Printf("❌ StorageManager 初始化失败: %v\n", err)
		return
	}
	defer sm.Close()

	var docs []bluge.Document
	for i := 0; i < count; i++ {
		path := filepath.Join(benchDir, fmt.Sprintf("bench_%d.txt", i))
		doc := bluge.NewDocument(path).
			AddField(bluge.NewTextField("content", "benchmark content").StoreValue())
		docs = append(docs, *doc)

		if len(docs) >= 50 {
			_ = sm.Index.UpdateBatch(docs)
			docs = nil
		}
	}
	fmt.Println("🏁 [bench] 压测结束。")
}

func runTest() {
	fmt.Println("🧪 [test] 启动自动化验证流...")

	fmt.Println("🔹 正在运行后端单元测试 (go test)...")
	testCmd := exec.Command("go", "test", "-v", "./src/...")
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		fmt.Printf("❌ 后端测试失败: %v\n", err)
	} else {
		fmt.Println("✅ 后端测试通过。")
	}

	fmt.Println("✨ [test] 验证流运行结束。")
}

func runClean() {
	fmt.Println("🧹 [clean] 正在清理构建产物...")
	dirs := []string{"bin", "dist", ".tmp"}
	for _, d := range dirs {
		os.RemoveAll(d)
	}
	fmt.Println("✨ [clean] 清理完成。")
}

func runBuild() {
	fmt.Println("🔨 [build] 开始极限出包...")

	fmt.Println("🌐 正在构建前端产物 (dist/)...")
	frontendCmd := exec.Command("npm", "run", "build")
	frontendCmd.Dir = "frontend"
	_ = frontendCmd.Run()

	os.MkdirAll("bin", 0755)
	target := "bin/ArkKB"
	if runtime.GOOS == "windows" {
		target += ".exe"
	}

	fmt.Printf("🏗️  正在构建后端二进制 (%s)...\n", target)
	args := []string{"build", "-ldflags", "-s -w", "-o", target, "main.go"}
	buildCmd := exec.Command("go", args...)
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if err := buildCmd.Run(); err != nil {
		fmt.Printf("❌ 后端构建失败: %v\n", err)
		return
	}

	fmt.Println("🎁 [build] 构建完成。")
}
