package captcha

import (
	"cdk-get/internal/config"
	"fmt"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// CaptchaPool 验证码客户端池
// 使用无锁轮询算法分配客户端
type CaptchaPool struct {
	clients []RemoteClient
	idx     atomic.Uint32
}

// NewCaptchaPool 从配置创建验证码客户端池
func NewCaptchaPool(providers []config.CaptchaProvider) (*CaptchaPool, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no captcha providers configured")
	}

	var clients []RemoteClient

	for i, provider := range providers {
		var client RemoteClient
		var err error

		switch provider.Type {
		case "ali":
			if provider.AccessKey == "" || provider.SecretKey == "" {
				logrus.Warnf("skipping ali provider at index %d: missing credentials", i)
				continue
			}
			client, err = NewAliCaptchaClient(provider.AccessKey, provider.SecretKey)
			if err != nil {
				logrus.Errorf("failed to create ali captcha client at index %d: %v", i, err)
				continue
			}
			logrus.Infof("successfully initialized ali captcha client")

		case "tencent":
			if provider.AccessKey == "" || provider.SecretKey == "" {
				logrus.Warnf("skipping tencent provider at index %d: missing credentials", i)
				continue
			}
			client, err = NewTcCaptchaClient(provider.AccessKey, provider.SecretKey)
			if err != nil {
				logrus.Errorf("failed to create tencent captcha client at index %d: %v", i, err)
				continue
			}
			logrus.Infof("successfully initialized tencent captcha client")

		case "google":
			if provider.CredentialsJSON == "" {
				logrus.Warnf("skipping google provider at index %d: missing credentials", i)
				continue
			}
			client, err = NewGoogleCaptchaClient(provider.CredentialsJSON)
			if err != nil {
				logrus.Errorf("failed to create google captcha client at index %d: %v", i, err)
				continue
			}
			logrus.Infof("successfully initialized google captcha client")

		default:
			logrus.Warnf("unknown captcha provider type at index %d: %s", i, provider.Type)
			continue
		}

		clients = append(clients, client)
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("failed to initialize any captcha clients")
	}

	logrus.Infof("captcha pool initialized with %d clients", len(clients))

	return &CaptchaPool{
		clients: clients,
	}, nil
}

// Get 获取下一个可用的验证码客户端
// 使用无锁轮询算法实现负载均衡
func (p *CaptchaPool) Get() RemoteClient {
	if len(p.clients) == 0 {
		return nil
	}

	// 原子递增索引并取模
	idx := p.idx.Add(1) - 1
	return p.clients[idx%uint32(len(p.clients))]
}

// Size 返回池中客户端数量
func (p *CaptchaPool) Size() int {
	return len(p.clients)
}
