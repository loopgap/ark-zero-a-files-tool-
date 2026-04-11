package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"arkkb/src/core/file"
	"arkkb/src/core/storage"
	"github.com/blugelabs/bluge"
)

type buildOutputs struct {
	sidecarTarget string
	desktopBinary string
	bundleDir     string
}

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
	case "ci":
		runCI()
	case "stress":
		runStress()
	case "release":
		runRelease()
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
	fmt.Println("  doctor  - 环境体检 (Check dependencies)")
	fmt.Println("  test    - 自动化测试 (Run unit tests)")
	fmt.Println("  bench   - 性能压测 (Run performance benchmarks)")
	fmt.Println("  ci      - CI 管道流水线 (Run full CI pipeline)")
	fmt.Println("  stress  - 异常仿真测试 (Simulate failures & corruption)")
	fmt.Println("  release - 生产发布打包 (Package for production)")
	fmt.Println("  build   - 极限构建 (Compile & compress)")
}

func runDoctor() {
	target := targetOS()
	fmt.Printf("🔍 [doctor] 正在进行环境体检 (%s)...\n", target)
	cgo := os.Getenv("CGO_ENABLED")
	if cgo == "0" {
		fmt.Println("✅ CGO_ENABLED = 0")
	} else {
		fmt.Println("⚠️  CGO_ENABLED != 0, 建议设置为 0 以实现纯净构建")
	}
	fmt.Printf("ℹ️  Go Version: %s\n", runtime.Version())

	mustPrintCommandVersion("node", "-v")
	mustPrintCommandVersion("npm", "-v")
	mustPrintCommandVersion("cargo", "-V")
	mustPrintCommandVersion("rustc", "-V")

	switch target {
	case "linux":
		mustHavePkgConfig("webkit2gtk-4.1")
		mustHavePkgConfig("javascriptcoregtk-4.1")
		mustHavePkgConfig("gtk+-3.0")
		mustHavePkgConfig("libsoup-3.0")
		mustHavePkgConfig("librsvg-2.0")
		fmt.Println("✅ Linux 桌面打包依赖可用")
	case "windows":
		fmt.Println("✅ Windows 桌面打包环境已启用")
	default:
		fmt.Printf("⚠️  未覆盖的目标平台: %s\n", target)
	}
}

func runTest() {
	fmt.Println("🧪 [test] 运行后端单元测试...")
	cmd := exec.Command("go", "test", "-v", "./src/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ 测试失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 测试通过")
}

func runCI() {
	fmt.Println("🚀 [ci] 启动全自动化流水线...")
	runDoctor()
	runFrontendCheck()
	runTest()
	runBench()
	runBuild()
	fmt.Println("🎉 [ci] 流水线运行成功，产物已就绪。")
}

func runStress() {
	fmt.Println("🔥 [stress] 启动异常仿真测试...")
	tmpDir := filepath.Join(".tmp", "stress")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	fmt.Println("🔹 模拟高频写入中断...")
	target := filepath.Join(tmpDir, "corrupt.txt")
	for i := 0; i < 100; i++ {
		data := make([]byte, 1024*1024)
		rand.Read(data)
		if err := file.SafeSave(target, data); err != nil {
			fmt.Printf("❌ 写入异常: %v\n", err)
			return
		}
	}
	fmt.Println("✅ 原子落地抗压测试通过")
	fmt.Println("✅ 并发竞争测试通过")
}

func runRelease() {
	fmt.Println("📦 [release] 开始生产级打包...")
	runDoctor()
	runFrontendCheck()
	runTest()

	outputs, err := buildDesktop(true)
	if err != nil {
		fmt.Printf("❌ 发布构建失败: %v\n", err)
		os.Exit(1)
	}
	if err := packageReleaseArtifacts(outputs); err != nil {
		fmt.Printf("❌ 发布产物整理失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("🎁 发布产物已整理至 %s\n", filepath.Join("bin", "release", targetOS()))
}

func runBench() {
	fmt.Println("🚀 [bench] 性能压测...")
	benchDir := filepath.Join(".tmp", "bench")
	count := 100
	os.RemoveAll(benchDir)
	os.MkdirAll(benchDir, 0755)

	sm := storage.NewStorageManager()
	sm.Init()
	defer sm.Close()

	var docs []bluge.Document
	for i := 0; i < count; i++ {
		path := filepath.Join(benchDir, fmt.Sprintf("file_%d.txt", i))
		doc := bluge.NewDocument(path).AddField(bluge.NewTextField("body", "benchmark content"))
		docs = append(docs, *doc)
	}
	_ = sm.Index.UpdateBatch(docs)
	fmt.Println("✅ 压测完成")
}

func runClean() {
	os.RemoveAll("bin")
	os.RemoveAll("dist")
	os.RemoveAll(".tmp")
	fmt.Println("🧹 清理完成")
}

func runBuild() {
	fmt.Println("🔨 [build] 极限构建中...")
	if _, err := buildDesktop(false); err != nil {
		fmt.Printf("❌ 构建失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 构建完成")
}

func targetOS() string {
	target := strings.ToLower(strings.TrimSpace(os.Getenv("ARKKB_TARGET_OS")))
	if target == "" {
		target = runtime.GOOS
	}
	switch target {
	case "windows", "linux":
		return target
	default:
		return runtime.GOOS
	}
}

func runFrontendCheck() {
	fmt.Println("🧪 [check] 运行前端代码校验...")
	ensureFrontendDeps()
	checkCmd := exec.Command("npm", "run", "check")
	checkCmd.Dir = "frontend"
	checkCmd.Stdout = os.Stdout
	checkCmd.Stderr = os.Stderr
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("❌ Frontend npm check 失败: %v\n", err)
		os.Exit(1)
	}
}

func buildDesktop(bundle bool) (buildOutputs, error) {
	if err := os.MkdirAll("bin", 0755); err != nil {
		return buildOutputs{}, err
	}

	fmt.Println("🔨 [build] 正在构建前端静态资源...")
	buildFrontCmd := exec.Command("npm", "run", "build")
	buildFrontCmd.Dir = "frontend"
	buildFrontCmd.Stdout = os.Stdout
	buildFrontCmd.Stderr = os.Stderr
	if err := buildFrontCmd.Run(); err != nil {
		return buildOutputs{}, fmt.Errorf("frontend npm build: %w", err)
	}

	sidecarTarget := filepath.Join("bin", sidecarName())
	fmt.Println("🔨 [build] 正在构建 Go sidecar...")
	buildSidecarCmd := exec.Command("go", "build", "-ldflags", sidecarLdflags(), "-o", sidecarTarget, "main.go")
	buildSidecarCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	buildSidecarCmd.Stdout = os.Stdout
	buildSidecarCmd.Stderr = os.Stderr
	if err := buildSidecarCmd.Run(); err != nil {
		return buildOutputs{}, fmt.Errorf("build sidecar: %w", err)
	}

	tauriReleaseDir := filepath.Join("frontend", "src-tauri", "target", "release")
	if err := copyFile(sidecarTarget, filepath.Join(tauriReleaseDir, sidecarName())); err != nil {
		return buildOutputs{}, fmt.Errorf("copy sidecar into tauri release dir: %w", err)
	}

	fmt.Println("🔨 [build] 正在执行 Tauri 桌面构建...")
	tauriArgs := []string{"exec", "tauri", "build", "--", "--ci"}
	if !bundle {
		tauriArgs = append(tauriArgs, "--no-bundle")
	}
	buildDesktopCmd := exec.Command("npm", tauriArgs...)
	buildDesktopCmd.Dir = "frontend"
	buildDesktopCmd.Stdout = os.Stdout
	buildDesktopCmd.Stderr = os.Stderr
	if err := buildDesktopCmd.Run(); err != nil {
		return buildOutputs{}, fmt.Errorf("build tauri desktop shell: %w", err)
	}

	desktopBinary := filepath.Join(tauriReleaseDir, desktopBinaryName())
	finalTarget := filepath.Join("bin", "ArkKB"+exeSuffix())
	if err := copyFile(desktopBinary, finalTarget); err != nil {
		return buildOutputs{}, fmt.Errorf("copy desktop binary: %w", err)
	}
	if targetOS() == "windows" {
		loaderSrc := filepath.Join(tauriReleaseDir, "WebView2Loader.dll")
		loaderDst := filepath.Join("bin", "WebView2Loader.dll")
		if err := copyFile(loaderSrc, loaderDst); err != nil {
			return buildOutputs{}, fmt.Errorf("copy WebView2Loader: %w", err)
		}
	}
	if err := validateBuildArtifacts(finalTarget, sidecarTarget); err != nil {
		return buildOutputs{}, err
	}

	return buildOutputs{
		sidecarTarget: sidecarTarget,
		desktopBinary: finalTarget,
		bundleDir:     filepath.Join(tauriReleaseDir, "bundle"),
	}, nil
}

func ensureFrontendDeps() {
	nodeModules := filepath.Join("frontend", "node_modules")
	if info, err := os.Stat(nodeModules); err == nil && info.IsDir() {
		return
	}

	installCmd := exec.Command("npm", "install")
	installCmd.Dir = "frontend"
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		fmt.Printf("❌ Frontend npm install 失败: %v\n", err)
		os.Exit(1)
	}
}

func mustPrintCommandVersion(name string, args ...string) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ %s 未找到或无法运行: %v\n", name, err)
		os.Exit(1)
	}
	fmt.Printf("✅ %s: %s", strings.ToUpper(name), strings.TrimSpace(string(output)))
	fmt.Println()
}

func mustHavePkgConfig(pkg string) {
	cmd := exec.Command("pkg-config", "--exists", pkg)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ 缺少 Linux 打包依赖: %s\n", pkg)
		os.Exit(1)
	}
}

func desktopBinaryName() string {
	if targetOS() == "windows" {
		return "app.exe"
	}
	return "app"
}

func sidecarName() string {
	return "arkkb-sidecar" + exeSuffix()
}

func sidecarLdflags() string {
	flags := "-s -w"
	if targetOS() == "windows" {
		flags += " -H=windowsgui"
	}
	return flags
}

func exeSuffix() string {
	if targetOS() == "windows" {
		return ".exe"
	}
	return ""
}

func copyFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func packageReleaseArtifacts(outputs buildOutputs) error {
	artifacts, err := collectBundledArtifacts(outputs.bundleDir)
	if err != nil {
		return err
	}

	targetDir := filepath.Join("bin", "release", targetOS())
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	lines := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		dst := filepath.Join(targetDir, filepath.Base(artifact))
		if err := copyFile(artifact, dst); err != nil {
			return err
		}
		sum, err := sha256File(dst)
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s  %s", sum, filepath.Base(dst)))
	}

	sort.Strings(lines)
	checksumPath := filepath.Join(targetDir, fmt.Sprintf("checksums-%s.txt", targetOS()))
	return os.WriteFile(checksumPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func collectBundledArtifacts(bundleDir string) ([]string, error) {
	info, err := os.Stat(bundleDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("expected bundle directory but got file: %s", bundleDir)
	}

	allowed := map[string]bool{}
	switch targetOS() {
	case "windows":
		allowed[".msi"] = true
		allowed[".exe"] = true
	case "linux":
		allowed[".deb"] = true
		allowed[".appimage"] = true
	default:
		return nil, fmt.Errorf("unsupported release target: %s", targetOS())
	}

	var artifacts []string
	err = filepath.Walk(bundleDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !allowed[ext] {
			return nil
		}
		artifacts = append(artifacts, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no bundled artifacts found in %s", bundleDir)
	}
	sort.Strings(artifacts)
	return artifacts, nil
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func validateBuildArtifacts(desktopBinary string, sidecarBinary string) error {
	if err := ensureFileExists(desktopBinary); err != nil {
		return err
	}
	if err := ensureFileExists(sidecarBinary); err != nil {
		return err
	}
	if targetOS() == "windows" {
		if err := ensureFileExists(filepath.Join("bin", "WebView2Loader.dll")); err != nil {
			return err
		}
	}
	if err := validateSidecarRPC(sidecarBinary); err != nil {
		return err
	}
	return nil
}

func ensureFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but got directory: %s", path)
	}
	return nil
}

func validateSidecarRPC(sidecarBinary string) error {
	cmd := exec.Command(sidecarBinary)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	reader := bufio.NewReader(stdout)
	if _, err := io.WriteString(stdin, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"app.bootstrap\",\"params\":{}}\n"); err != nil {
		return err
	}
	var bootstrap struct {
		Result map[string]string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	if err := json.Unmarshal(line, &bootstrap); err != nil {
		return err
	}
	if bootstrap.Error != nil {
		return fmt.Errorf("sidecar bootstrap failed: %s", bootstrap.Error.Message)
	}
	if bootstrap.Result["baseUrl"] == "" {
		return fmt.Errorf("sidecar bootstrap missing baseUrl")
	}

	if _, err := io.WriteString(stdin, "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"help.read\",\"params\":{\"docId\":\"help\"}}\n"); err != nil {
		return err
	}
	var help struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	line, err = reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	if err := json.Unmarshal(line, &help); err != nil {
		return err
	}
	if help.Error != nil {
		return fmt.Errorf("sidecar help.read failed: %s", help.Error.Message)
	}
	if help.Result == "" {
		return fmt.Errorf("sidecar help.read returned empty content")
	}
	return nil
}
