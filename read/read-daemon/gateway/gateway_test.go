package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGateway(t *testing.T) (*httptest.Server, string) {
	wsDir, err := os.MkdirTemp("", "gateway_test_ws_*")
	if err != nil {
		t.Fatalf("failed to create temp ws: %v", err)
	}

	// Create a test file
	_ = os.WriteFile(filepath.Join(wsDir, "test.txt"), []byte("hello gateway"), 0644)

	// Create symlink pointing outside to /etc
	_ = os.Symlink("/etc", filepath.Join(wsDir, "symlink_etc"))

	cfg := ServerConfig{
		Port:          0,
		WorkspaceRoot: wsDir,
		ReadEngineBin: "/bin/echo",
	}

	server, err := NewGatewayServer(cfg)
	if err != nil {
		os.RemoveAll(wsDir)
		t.Fatalf("failed to create gateway server: %v", err)
	}

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	return ts, wsDir
}

func TestGatewayFsTraversalRejection(t *testing.T) {
	ts, wsDir := setupTestGateway(t)
	defer ts.Close()
	defer os.RemoveAll(wsDir)

	client := ts.Client()

	traversalPayloads := []string{
		"../etc/passwd",
		"../../../../etc/shadow",
		"%2e%2e%2fetc/passwd",
		"%2e%2e/%2e%2e/etc/passwd",
		"%252e%252e%252fetc/passwd",
		"symlink_etc/passwd",
		"symlink_etc/hosts",
		"symlink_etc",
		"/etc/passwd",
	}

	// 1. Verify GET /api/v1/fs/read returns 403 Forbidden on all traversal payloads
	for _, payload := range traversalPayloads {
		resp, err := client.Get(ts.URL + "/api/v1/fs/read?path=" + payload)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("expected 403 Forbidden for path %q, got: %d (body: %s)", payload, resp.StatusCode, string(body))
		}
		resp.Body.Close()
	}

	// 2. Verify GET /api/v1/fs/list returns 403 Forbidden on traversal payloads
	for _, payload := range []string{"../", "../../", "%2e%2e%2f", "symlink_etc"} {
		resp, err := client.Get(ts.URL + "/api/v1/fs/list?path=" + payload)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			resp.Body.Close()
			t.Fatalf("expected 403 Forbidden for list path %q, got: %d", payload, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 3. Verify POST /api/v1/fs/write returns 403 Forbidden on traversal payloads
	for _, payload := range traversalPayloads {
		resp, err := client.Post(ts.URL+"/api/v1/fs/write?path="+payload, "text/plain", bytes.NewBufferString("malicious"))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			resp.Body.Close()
			t.Fatalf("expected 403 Forbidden for write path %q, got: %d", payload, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 4. Verify POST /api/v1/fs/delete returns 403 Forbidden on traversal payloads
	for _, payload := range []string{"../etc/passwd", "symlink_etc", "%2e%2e%2fetc/passwd"} {
		resp, err := client.Post(ts.URL+"/api/v1/fs/delete?path="+payload, "application/json", nil)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			resp.Body.Close()
			t.Fatalf("expected 403 Forbidden for delete path %q, got: %d", payload, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 5. Verify valid read returns 200 OK
	validResp, err := client.Get(ts.URL + "/api/v1/fs/read?path=test.txt")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer validResp.Body.Close()
	if validResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for valid file, got: %d", validResp.StatusCode)
	}
	body, _ := io.ReadAll(validResp.Body)
	if string(body) != "hello gateway" {
		t.Fatalf("content mismatch: got %q", string(body))
	}
}

func TestGatewayTokenRedactionInTaskCommand(t *testing.T) {
	ts, wsDir := setupTestGateway(t)
	defer ts.Close()
	defer os.RemoveAll(wsDir)

	client := ts.Client()

	// Call /api/v1/task/command with sensitive key in args
	payload := map[string]interface{}{
		"command": "OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890",
		"args":    []string{"ANTHROPIC_API_KEY=sk-ant-api03-abcdef1234567890abcdef"},
	}
	data, _ := json.Marshal(payload)

	resp, err := client.Post(ts.URL+"/api/v1/task/command", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	output, ok := res["output"].(string)
	if !ok {
		t.Fatalf("missing output field in response")
	}

	// Verify tokens are masked
	if strings.Contains(output, "sk-proj-") || strings.Contains(output, "sk-ant-") {
		t.Fatalf("unmasked token leaked in task/command output: %s", output)
	}
	if !strings.Contains(output, "OPENAI_API_KEY=***REDACTED***") {
		t.Fatalf("expected OPENAI_API_KEY=***REDACTED*** in output, got: %s", output)
	}
}

func TestGatewayPipelineRunEndpoint(t *testing.T) {
	ts, wsDir := setupTestGateway(t)
	defer ts.Close()
	defer os.RemoveAll(wsDir)

	client := ts.Client()

	// 1. GET method should be rejected with 405 Method Not Allowed
	getResp, err := client.Get(ts.URL + "/api/v1/pipeline/run")
	if err != nil {
		t.Fatalf("GET /api/v1/pipeline/run failed: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got: %d", getResp.StatusCode)
	}

	// 2. Empty prompt should be rejected with 400 Bad Request
	emptyReq := map[string]interface{}{"prompt": ""}
	emptyData, _ := json.Marshal(emptyReq)
	emptyResp, err := client.Post(ts.URL+"/api/v1/pipeline/run", "application/json", bytes.NewReader(emptyData))
	if err != nil {
		t.Fatalf("POST /api/v1/pipeline/run failed: %v", err)
	}
	defer emptyResp.Body.Close()
	if emptyResp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty prompt, got: %d", emptyResp.StatusCode)
	}
}

func TestGatewayBearerTokenAuth(t *testing.T) {
	wsDir, err := os.MkdirTemp("", "gateway_auth_test_*")
	if err != nil {
		t.Fatalf("failed to create temp ws: %v", err)
	}
	defer os.RemoveAll(wsDir)

	token := "read_auth_secret_998877"
	cfg := ServerConfig{
		Port:          0,
		WorkspaceRoot: wsDir,
		ReadEngineBin: "/bin/echo",
		AuthToken:     token,
	}

	server, err := NewGatewayServer(cfg)
	if err != nil {
		t.Fatalf("failed to create gateway server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	client := ts.Client()

	// 1. /api/v1/status should succeed without auth (public health check)
	statusResp, err := client.Get(ts.URL + "/api/v1/status")
	if err != nil {
		t.Fatalf("status request failed: %v", err)
	}
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for unauthenticated /api/v1/status, got %d", statusResp.StatusCode)
	}
	statusResp.Body.Close()

	// 2. /api/v1/harnesses without token should return 401 Unauthorized
	unauthResp, err := client.Get(ts.URL + "/api/v1/harnesses")
	if err != nil {
		t.Fatalf("unauth request failed: %v", err)
	}
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without token, got %d", unauthResp.StatusCode)
	}
	unauthResp.Body.Close()

	// 3. /api/v1/harnesses with wrong token should return 401 Unauthorized
	reqWrong, _ := http.NewRequest("GET", ts.URL+"/api/v1/harnesses", nil)
	reqWrong.Header.Set("Authorization", "Bearer invalid_secret")
	wrongResp, err := client.Do(reqWrong)
	if err != nil {
		t.Fatalf("wrong token request failed: %v", err)
	}
	if wrongResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized with invalid token, got %d", wrongResp.StatusCode)
	}
	wrongResp.Body.Close()

	// 4. /api/v1/harnesses with correct token should return 200 OK
	reqValid, _ := http.NewRequest("GET", ts.URL+"/api/v1/harnesses", nil)
	reqValid.Header.Set("Authorization", "Bearer "+token)
	validResp, err := client.Do(reqValid)
	if err != nil {
		t.Fatalf("valid token request failed: %v", err)
	}
	if validResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK with valid token, got %d", validResp.StatusCode)
	}
	validResp.Body.Close()
}

func TestGatewayRateLimitingAndWhitelist(t *testing.T) {
	wsDir, err := os.MkdirTemp("", "gateway_ratelimit_test_*")
	if err != nil {
		t.Fatalf("failed to create temp ws: %v", err)
	}
	defer os.RemoveAll(wsDir)

	cfg := ServerConfig{
		Port:          0,
		WorkspaceRoot: wsDir,
		ReadEngineBin: "/bin/echo",
	}

	server, err := NewGatewayServer(cfg)
	if err != nil {
		t.Fatalf("failed to create gateway server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	client := ts.Client()

	// 1. Localhost whitelist verification: 25 requests from loopback MUST all succeed (no 429)
	for i := 0; i < 25; i++ {
		resp, err := client.Get(ts.URL + "/api/v1/status")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for loopback request %d, got %d", i, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 2. External IP rate limiting: simulate client IP 198.51.100.42
	externalIP := "198.51.100.42"
	var lastStatus int
	var retryAfterHeader string

	for i := 0; i < 15; i++ {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/status", nil)
		req.Header.Set("X-Forwarded-For", externalIP)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("external request %d failed: %v", i, err)
		}
		lastStatus = resp.StatusCode
		retryAfterHeader = resp.Header.Get("Retry-After")
		resp.Body.Close()

		if i < 10 {
			if lastStatus != http.StatusOK {
				t.Fatalf("expected 200 for request %d within burst, got %d", i, lastStatus)
			}
		} else {
			// Burst exceeded (requests 10-14 should trigger 429)
			if lastStatus == http.StatusTooManyRequests {
				break
			}
		}
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("expected 429 StatusTooManyRequests after exceeding burst, got %d", lastStatus)
	}
	if retryAfterHeader == "" {
		t.Fatalf("expected non-empty Retry-After header on 429 response")
	}
}
