package httpclient

import (
	"net"
	"net/http"
	"time"
)

// ClientConfig HTTP客户端配置
type ClientConfig struct {
	Timeout               time.Duration // 总超时时间
	DialTimeout           time.Duration // 连接超时时间
	KeepAlive             time.Duration // Keep-Alive 时间
	MaxIdleConns          int           // 最大空闲连接数
	MaxIdleConnsPerHost   int           // 每个主机最大空闲连接数
	IdleConnTimeout       time.Duration // 空闲连接超时时间
	TLSHandshakeTimeout   time.Duration // TLS握手超时时间
	ExpectContinueTimeout time.Duration // Expect: 100-continue 超时时间
}

// DefaultConfig 返回默认配置
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Timeout:               30 * time.Second,
		DialTimeout:           20 * time.Second,
		KeepAlive:             30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       20 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
	}
}

// ClientFactory HTTP客户端工厂
type ClientFactory struct {
	config ClientConfig
}

// NewClientFactory 创建客户端工厂
func NewClientFactory(config ClientConfig) *ClientFactory {
	return &ClientFactory{
		config: config,
	}
}

// NewClient 创建新的HTTP客户端
func (f *ClientFactory) NewClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   f.config.DialTimeout,
				KeepAlive: f.config.KeepAlive,
			}).DialContext,
			MaxIdleConns:          f.config.MaxIdleConns,
			MaxIdleConnsPerHost:   f.config.MaxIdleConnsPerHost,
			IdleConnTimeout:       f.config.IdleConnTimeout,
			TLSHandshakeTimeout:   f.config.TLSHandshakeTimeout,
			ExpectContinueTimeout: f.config.ExpectContinueTimeout,
		},
		Timeout: f.config.Timeout,
	}
}

// NewDefaultClient 使用默认配置创建HTTP客户端
func NewDefaultClient() *http.Client {
	factory := NewClientFactory(DefaultConfig())
	return factory.NewClient()
}
