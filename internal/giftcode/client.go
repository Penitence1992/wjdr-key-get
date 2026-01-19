package giftcode

import (
	"cdk-get/internal/captcha"
	"cdk-get/internal/storage"
	"cdk-get/internal/utls"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const ddSecretKey = "Uiv#87#SPan.ECsp"

const ErrMsgRetryMsg = "TIMEOUT RETRY"

const ErrMsgReceived = "RECEIVED."
const ErrMsgCdkNotFound = "CDK NOT FOUND."

type DdResult struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type DdPlayerMsg struct {
	DdResult
	ErrCode string `json:"err_code"`
	Data    struct {
		Fid      int    `json:"fid"`
		Nickname string `json:"nickname"`
		Kid      int    `json:"kid"`
		Avatar   string `json:"avatar_image"`
	} `json:"data"`
}

type DdImgMsg struct {
	DdResult
	ErrCode int `json:"err_code"`
	Data    struct {
		Img string `json:"img"`
	} `json:"data"`
}

type PlayerGiftCode struct {
	Fid           string
	init          bool
	expireTime    time.Time
	Player        *DdPlayerMsg
	clientFn      func() captcha.RemoteClient
	storageCli    storage.KeyStorage
	correlationID string // 用于日志关联
}

func NewPlayerGiftCode(fid string, clientFn func() captcha.RemoteClient, storageCli storage.KeyStorage) *PlayerGiftCode {
	return &PlayerGiftCode{
		Fid:           fid,
		clientFn:      clientFn,
		storageCli:    storageCli,
		correlationID: generateCorrelationID(),
	}
}

func (g *PlayerGiftCode) GetGift(code string) (result *DdResult, err error) {
	ctx := context.Background()
	return g.GetGiftWithContext(ctx, code)
}

func (g *PlayerGiftCode) GetGiftWithContext(ctx context.Context, code string) (result *DdResult, err error) {
	log := logrus.WithFields(logrus.Fields{
		"operation":      "get_gift",
		"fid":            g.Fid,
		"code":           code,
		"correlation_id": g.correlationID,
	})

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before getting gift: %w", ctx.Err())
	default:
	}

	if !g.init {
		if err = g.doInitWithContext(ctx); err != nil {
			log.WithError(err).Error("failed to initialize player")
			return nil, fmt.Errorf("failed to initialize player: %w", err)
		}
		g.init = true
	} else {
		if time.Now().After(g.expireTime) {
			if err = g.doInitWithContext(ctx); err != nil {
				log.WithError(err).Error("failed to re-initialize player")
				return nil, fmt.Errorf("failed to re-initialize player: %w", err)
			}
			g.init = true
		}
	}

	params := url.Values{}
	if imgResp, err := g.getCaptchaWithContext(ctx); err != nil {
		log.WithError(err).Error("failed to get captcha")
		return nil, fmt.Errorf("failed to get captcha: %w", err)
	} else {
		if captchaImg, err := g.clientFn().DoWithBase64Img(imgResp.Data.Img); err != nil {
			log.WithError(err).Error("failed to recognize captcha")
			return nil, fmt.Errorf("failed to recognize captcha: %w", err)
		} else {
			if captchaImg.Content == "" {
				log.WithField("word", captchaImg.Word).Warn("captcha recognition failed")
				return nil, fmt.Errorf("验证码识别失败，错误信息：%s", captchaImg.Word)
			} else {
				params.Add("captcha_code", strings.TrimSpace(captchaImg.Content))
				log.WithField("captcha_code", strings.TrimSpace(captchaImg.Content)).Debug("captcha recognized")
			}
		}
	}

	params.Add("fid", g.Fid)
	params.Add("time", fmt.Sprintf("%d", time.Now().UnixMilli()))
	params.Add("cdk", code)

	result, err = utls.SendRequestV2[DdResult]("gift_code", params, ddSecretKey)
	if err != nil {
		log.WithError(err).Error("failed to send gift code request")
		return nil, fmt.Errorf("failed to send gift code request: %w", err)
	}

	log.WithFields(logrus.Fields{
		"result_code": result.Code,
		"result_msg":  result.Msg,
	}).Info("gift code request completed")

	return
}

func (g *PlayerGiftCode) getCaptcha() (result *DdImgMsg, err error) {
	ctx := context.Background()
	return g.getCaptchaWithContext(ctx)
}

func (g *PlayerGiftCode) getCaptchaWithContext(ctx context.Context) (result *DdImgMsg, err error) {
	log := logrus.WithFields(logrus.Fields{
		"operation":      "get_captcha",
		"fid":            g.Fid,
		"correlation_id": g.correlationID,
	})

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before getting captcha: %w", ctx.Err())
	default:
	}

	params := url.Values{}
	params.Add("fid", g.Fid)
	params.Add("time", fmt.Sprintf("%d", time.Now().UnixMilli()))
	params.Add("init", "0")

	result, err = utls.SendRequestV2[DdImgMsg]("captcha", params, ddSecretKey)
	if err != nil {
		log.WithError(err).Error("failed to send captcha request")
		return nil, fmt.Errorf("failed to send captcha request: %w", err)
	}
	if result == nil {
		log.Error("captcha response is nil")
		return nil, fmt.Errorf("无法获取验证码")
	}
	if result.Code != 0 {
		log.WithFields(logrus.Fields{
			"code": result.Code,
			"msg":  result.Msg,
		}).Error("captcha request failed")
		return nil, fmt.Errorf("获取验证码失败，错误信息：%s", result.Msg)
	}

	log.Debug("captcha retrieved successfully")
	return result, nil
}

func (g *PlayerGiftCode) doInit() error {
	ctx := context.Background()
	return g.doInitWithContext(ctx)
}

func (g *PlayerGiftCode) doInitWithContext(ctx context.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"operation":      "init_player",
		"fid":            g.Fid,
		"correlation_id": g.correlationID,
	})

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before initializing player: %w", ctx.Err())
	default:
	}

	params := url.Values{}
	params.Add("fid", g.Fid)
	params.Add("time", fmt.Sprintf("%d", time.Now().UnixMilli()))

	player, err := utls.SendRequestV2[DdPlayerMsg]("player", params, ddSecretKey)
	if err != nil {
		log.WithError(err).Error("failed to get player info")
		return fmt.Errorf("failed to get player info: %w", err)
	}
	if player == nil {
		log.Error("player response is nil")
		return fmt.Errorf("无法获取用户信息")
	}

	g.Player = player
	g.expireTime = time.Now().Add(10 * time.Minute)
	d := g.Player.Data

	if err = g.storageCli.SaveFidInfo(d.Fid, d.Nickname, d.Kid, d.Avatar); err != nil {
		log.WithError(err).Error("failed to save player info")
		return fmt.Errorf("failed to save player info: %w", err)
	}

	log.WithFields(logrus.Fields{
		"nickname": d.Nickname,
		"kid":      d.Kid,
	}).Info("player initialized successfully")

	return nil
}

func (g *PlayerGiftCode) Init() error {
	if err := g.doInit(); err != nil {
		return err
	} else {
		g.init = true
		return nil
	}
}

func (g *PlayerGiftCode) InitWithContext(ctx context.Context) error {
	if err := g.doInitWithContext(ctx); err != nil {
		return err
	} else {
		g.init = true
		return nil
	}
}

// generateCorrelationID 生成唯一的关联ID用于日志追踪
func generateCorrelationID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
