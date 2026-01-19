package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// WxPusherNotifier implements Notifier for WxPusher
type WxPusherNotifier struct {
	appToken string
	uid      string
	client   *http.Client
	logger   *logrus.Logger
}

// wxPusherRequest represents the WxPusher API request body
type wxPusherRequest struct {
	AppToken      string   `json:"appToken"`
	Content       string   `json:"content"`
	Summary       string   `json:"summary"`
	Title         string   `json:"title"`
	ContentType   int      `json:"contentType"`
	UIDs          []string `json:"uids"`
	VerifyPay     bool     `json:"verifyPay"`
	VerifyPayType int      `json:"verifyPayType"`
}

// wxPusherResponse represents the WxPusher API response
type wxPusherResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    any    `json:"data"`
	Success bool   `json:"success"`
}

// NewWxPusherNotifier creates a new WxPusher notifier
func NewWxPusherNotifier(appToken, uid string, logger *logrus.Logger) *WxPusherNotifier {
	return &WxPusherNotifier{
		appToken: appToken,
		uid:      uid,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   20 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       20 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 3 * time.Second,
			},
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Send implements Notifier.Send
func (w *WxPusherNotifier) Send(ctx context.Context, req NotificationRequest) (*NotificationResult, error) {
	// Prepare request body
	body := wxPusherRequest{
		AppToken:      w.appToken,
		Content:       req.Content,
		Summary:       req.Summary,
		Title:         req.Title,
		ContentType:   2, // HTML content type
		UIDs:          []string{w.uid},
		VerifyPay:     false,
		VerifyPayType: 0,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		w.logger.WithError(err).Error("failed to marshal WxPusher request")
		return &NotificationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to marshal request: %v", err),
			Error:   err,
		}, err
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://wxpusher.zjiecode.com/api/send/message", bytes.NewBuffer(jsonBody))
	if err != nil {
		w.logger.WithError(err).Error("failed to create WxPusher HTTP request")
		return &NotificationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Error:   err,
		}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request with retry logic
	var lastErr error
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := w.client.Do(httpReq)
		if err != nil {
			lastErr = err
			w.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"error":   err.Error(),
			}).Warn("WxPusher request failed, retrying...")

			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
				continue
			}
			break
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			w.logger.WithError(err).Error("failed to read WxPusher response")
			return &NotificationResult{
				Success: false,
				Message: fmt.Sprintf("Failed to read response: %v", err),
				Error:   err,
			}, err
		}

		// Check HTTP status code
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
			w.logger.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
				"response":    string(respBody),
			}).Error("WxPusher API returned error status")
			return &NotificationResult{
				Success: false,
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
				Error:   lastErr,
			}, lastErr
		}

		// Parse response
		var wxResp wxPusherResponse
		if err := json.Unmarshal(respBody, &wxResp); err != nil {
			w.logger.WithError(err).WithField("response", string(respBody)).Warn("failed to parse WxPusher response, but request succeeded")
			// Consider it a success if we got a 2xx status code
			return &NotificationResult{
				Success: true,
				Message: string(respBody),
				Error:   nil,
			}, nil
		}

		// Check API response code
		if wxResp.Code != 1000 && !wxResp.Success {
			lastErr = fmt.Errorf("WxPusher API error: code=%d, msg=%s", wxResp.Code, wxResp.Msg)
			w.logger.WithFields(logrus.Fields{
				"code":    wxResp.Code,
				"message": wxResp.Msg,
			}).Error("WxPusher API returned error code")
			return &NotificationResult{
				Success: false,
				Message: fmt.Sprintf("API error: %s (code: %d)", wxResp.Msg, wxResp.Code),
				Error:   lastErr,
			}, lastErr
		}

		// Success
		w.logger.WithFields(logrus.Fields{
			"title":   req.Title,
			"message": wxResp.Msg,
		}).Info("notification sent successfully via WxPusher")

		return &NotificationResult{
			Success: true,
			Message: wxResp.Msg,
			Error:   nil,
		}, nil
	}

	// All retries failed
	w.logger.WithError(lastErr).Error("all WxPusher retry attempts failed")
	return &NotificationResult{
		Success: false,
		Message: fmt.Sprintf("All retry attempts failed: %v", lastErr),
		Error:   lastErr,
	}, lastErr
}

// GetChannel implements Notifier.GetChannel
func (w *WxPusherNotifier) GetChannel() string {
	return "wxpusher"
}
