package utls

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const browserUa string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   20 * time.Second, // 连接超时时间
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       20 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 3 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
)

// 生成签名（模拟原JS逻辑）
func generateSign(params url.Values, secretKey string) string {
	// 1. 按参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 拼接键值对
	var sb strings.Builder
	for _, k := range keys {
		if sb.Len() > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params.Get(k)) // 简单值直接拼接
	}

	// 3. 追加密钥
	signString := sb.String() + secretKey

	// 4. 计算MD5
	hash := md5.Sum([]byte(signString))
	return hex.EncodeToString(hash[:])
}

// 发送POST请求
func SendRequestV2[T any](path string, params url.Values, secretKey string) (*T, error) {
	// 生成签名并添加到参数
	signature := generateSign(params, secretKey)
	params.Add("sign", signature) // 根据实际字段名调整

	// 创建请求
	req, err := http.NewRequest("POST", fmt.Sprintf("https://wjdr-giftcode-api.campfiregames.cn/api/%s", path), strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Referer", "https://wjdr-giftcode.centurygames.cn/")
	req.Header.Add("User-Agent", browserUa)
	// 发送请求
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 反序列化
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func PushWithWxPusher(title, summary, content string) {
	appToken := "AT_gVv4hK5XwHo4nHdyFafikJxt2eIGFSDa"
	uid := "UID_Zgs55UrCKYtZnxXW3i9D7KQKCBhX"
	body := map[string]interface{}{
		"appToken":      appToken,
		"content":       content,
		"summary":       summary,
		"title":         title,
		"contentType":   2,
		"uids":          []string{uid},
		"verifyPay":     false,
		"verifyPayType": 0,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://wxpusher.zjiecode.com/api/send/message", bytes.NewBuffer(jsonBody))

	req.Header.Add("Content-Type", "application/json")

	resp, err := defaultClient.Do(req)
	if err != nil {
		logrus.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		return
	}
	logrus.Infof("通知成功: %s", string(respBody))
}
