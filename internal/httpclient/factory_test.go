package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 100 {
		t.Errorf("expected max idle conns 100, got %d", config.MaxIdleConns)
	}
}

func TestNewClientFactory(t *testing.T) {
	config := ClientConfig{
		Timeout:      10 * time.Second,
		DialTimeout:  5 * time.Second,
		MaxIdleConns: 50,
	}

	factory := NewClientFactory(config)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	if factory.config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", factory.config.Timeout)
	}
}

func TestClientFactory_NewClient(t *testing.T) {
	factory := NewClientFactory(DefaultConfig())
	client := factory.NewClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout != 30*time.Second {
		t.Errorf("expected client timeout 30s, got %v", client.Timeout)
	}

	if client.Transport == nil {
		t.Fatal("expected non-nil transport")
	}
}

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", client.Timeout)
	}
}

func TestClientCanMakeRequests(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// 使用工厂创建客户端
	client := NewDefaultClient()

	// 发送请求
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != "test response" {
		t.Errorf("expected 'test response', got %s", string(body))
	}
}

func TestClientTimeout(t *testing.T) {
	// 创建一个慢速服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 创建超时时间很短的客户端
	config := ClientConfig{
		Timeout:      100 * time.Millisecond,
		DialTimeout:  5 * time.Second,
		MaxIdleConns: 10,
	}
	factory := NewClientFactory(config)
	client := factory.NewClient()

	// 发送请求，应该超时
	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestClientConnectionReuse(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewDefaultClient()

	// 发送多个请求
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		io.ReadAll(resp.Body) // 读取body以便连接可以被复用
		resp.Body.Close()
	}

	if requestCount != 5 {
		t.Errorf("expected 5 requests, got %d", requestCount)
	}
}
