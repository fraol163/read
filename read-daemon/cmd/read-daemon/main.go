package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"read-daemon/gateway"
	"read-daemon/orchestrator"
	"read-daemon/scanner"
	"read-daemon/security"
)

func runCli(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	harnessFlag := fs.String("harness", "", "Harness ID to execute (e.g. dsh, claude, aider)")
	autoApproveFlag := fs.Bool("auto-approve", false, "Automatically apply Master Override if verification gate fails")
	timeoutFlag := fs.Int("timeout", 900, "Hard timeout limit in seconds")
	testCmdFlag := fs.String("test-cmd", "", "Custom test suite command")
	workspaceFlag := fs.String("workspace", ".", "Target repository workspace root")
	engineBinFlag := fs.String("engine-bin", "", "Path to native read-engine binary")
	customHarnessFlag := fs.String("custom-harness", "", "Path to custom harnesses.json")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	promptArgs := fs.Args()
	if len(promptArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Prompt argument required: read run [options] \"task prompt\"")
		os.Exit(1)
	}
	prompt := strings.Join(promptArgs, " ")

	engineBin := resolveEngineBin(*engineBinFlag)

	cfg := orchestrator.PipelineConfig{
		Workspace:           *workspaceFlag,
		Prompt:              prompt,
		HarnessID:           *harnessFlag,
		AutoApprove:         *autoApproveFlag,
		TimeoutSeconds:      *timeoutFlag,
		EngineBin:           engineBin,
		TestCommand:         *testCmdFlag,
		CustomHarnessConfig: *customHarnessFlag,
		Stdout:              os.Stdout,
		Stderr:              os.Stderr,
	}

	orch, err := orchestrator.NewOrchestrator(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to initialize orchestrator: %v\n", err)
		os.Exit(1)
	}

	res, err := orch.Run(context.Background())
	if err != nil || !res.Success {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Pipeline failed: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[ERROR] Pipeline verification gate failed: %s\n", res.Diagnostics)
		}
		os.Exit(1)
	}

	os.Exit(0)
}

func scanCli(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	customConfig := fs.String("custom-harness", "", "Path to custom harnesses.json")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	reg, err := scanner.ScanHost(*customConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Scan failed: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		data, _ := json.MarshalIndent(reg, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Printf("READ Framework Harness Scanner (v0.1.0)\n")
	fmt.Printf("Detected %d AI agent harnesses / servers on host:\n", len(reg.Harnesses))
	fmt.Println("────────────────────────────────────────────────────────────────")
	for _, h := range reg.Harnesses {
		fmt.Printf("- [%s] %-34s (%s)\n    Status:  %-12s Version: %s\n    Path:    %s\n",
			h.Type, h.Name, h.ID, h.Status, h.Version, h.Path)
	}
	fmt.Println("────────────────────────────────────────────────────────────────")
	os.Exit(0)
}

func resolveEngineBin(specified string) string {
	// 1. If specified binary exists on disk, use its absolute path
	if specified != "" {
		if abs, err := filepath.Abs(specified); err == nil {
			if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
				return abs
			}
		}
	}

	// 2. Check sibling binary next to read-daemon executable
	if exePath, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exePath), "read-engine")
		if fi, err := os.Stat(sibling); err == nil && !fi.IsDir() {
			return sibling
		}
	}

	// 3. Check PATH
	if p, err := exec.LookPath("read-engine"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}

	// 4. Default fallback
	abs, _ := filepath.Abs(specified)
	return abs
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		runCli(os.Args[2:])
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "scan" {
		scanCli(os.Args[2:])
		return
	}

	host := flag.String("host", "", "Host address to bind (default: 127.0.0.1; 0.0.0.0 or public interface requires TLS)")
	port := flag.Int("port", 8089, "Gateway port for remote client connection")
	workspace := flag.String("workspace", ".", "Target repository workspace root")
	customConfig := flag.String("custom-harness", "", "Path to custom harnesses.json")
	engineBin := flag.String("engine-bin", "./read/bin/read-engine", "Path to native read-engine binary")
	versionFlag := flag.Bool("version", false, "Print version information")
	printRlimitFlag := flag.Bool("print-rlimit", false, "Print dynamically calibrated RLIMIT_AS in bytes")
	meminfoPath := flag.String("meminfo", "", "Custom meminfo file path for dynamic memory testing")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file")
	authToken := flag.String("auth-token", "", "Shared secret Bearer authentication token")
	flag.Parse()

	if *versionFlag {
		fmt.Println("READ Framework Host Daemon (v0.1.0)")
		return
	}

	if *printRlimitFlag {
		var limit uint64
		if *meminfoPath != "" {
			limit = security.CalibrateMemoryLimitFrom(*meminfoPath)
		} else {
			limit = security.CalibrateMemoryLimit()
		}
		fmt.Printf("%d\n", limit)
		return
	}

	hasTLS := *tlsCert != "" || *tlsKey != ""
	if hasTLS && (*tlsCert == "" || *tlsKey == "") {
		log.Fatalf("[SECURITY VIOLATION] Both --tls-cert and --tls-key must be provided for TLS operation")
	}

	targetHost := *host
	if targetHost == "" {
		if hasTLS {
			targetHost = "0.0.0.0"
		} else {
			targetHost = "127.0.0.1"
		}
	} else {
		isLoopback := targetHost == "127.0.0.1" || targetHost == "localhost" || targetHost == "::1" || strings.HasPrefix(targetHost, "127.")
		if !hasTLS && !isLoopback {
			log.Fatalf("[SECURITY VIOLATION] Mechanically rejected: Binding to non-loopback address %s is strictly forbidden without TLS (--tls-cert and --tls-key required).", targetHost)
		}
	}

	if hasTLS {
		if _, err := os.Stat(*tlsCert); err != nil {
			log.Fatalf("[ERROR] TLS certificate file not found: %v", err)
		}
		if _, err := os.Stat(*tlsKey); err != nil {
			log.Fatalf("[ERROR] TLS private key file not found: %v", err)
		}
	}

	absWorkspace, err := filepath.Abs(*workspace)
	if err != nil {
		log.Fatalf("Invalid workspace: %v", err)
	}

	absEngine := resolveEngineBin(*engineBin)

	cfg := gateway.ServerConfig{
		Host:          targetHost,
		Port:          *port,
		WorkspaceRoot: absWorkspace,
		CustomHarness: *customConfig,
		ReadEngineBin: absEngine,
		TLSCert:       *tlsCert,
		TLSKey:        *tlsKey,
		AuthToken:     *authToken,
	}

	server, err := gateway.NewGatewayServer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize gateway server: %v", err)
	}

	handler := server.Handler()

	dynamicLimit := security.GetDynamicMemoryLimit()
	addr := fmt.Sprintf("%s:%d", targetHost, *port)
	scheme := "http"
	if hasTLS {
		scheme = "https"
	}

	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Printf("READ Framework Host Daemon (v0.1.0)\n")
	fmt.Printf("Listening on:   %s://%s\n", scheme, addr)
	fmt.Printf("Workspace:      %s\n", absWorkspace)
	fmt.Printf("Native Core:    %s\n", absEngine)
	fmt.Printf("Dynamic RLIMIT: %d MB (30%% available RAM ceiling)\n", dynamicLimit/(1024*1024))
	if hasTLS {
		fmt.Printf("Security:       TLS/HTTPS ENFORCED\n")
	} else {
		fmt.Printf("Security:       LOOPBACK ONLY (TLS Disabled)\n")
	}
	if *authToken != "" {
		fmt.Printf("Auth:           TOKEN AUTH ENFORCED\n")
	} else {
		fmt.Printf("Auth:           OPEN (Loopback dev mode)\n")
	}
	fmt.Println("Ready for remote controller connections...")
	fmt.Println("────────────────────────────────────────────────────────────────")

	if hasTLS {
		if err := http.ListenAndServeTLS(addr, *tlsCert, *tlsKey, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		if err := http.ListenAndServe(addr, handler); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}
}
