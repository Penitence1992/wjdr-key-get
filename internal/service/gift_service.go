package service

import (
	"cdk-get/internal/cache"
	"cdk-get/internal/captcha"
	"cdk-get/internal/errors"
	"cdk-get/internal/giftcode"
	"cdk-get/internal/storage"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RedeemResult 兑换结果
type RedeemResult struct {
	FID      string
	Code     string
	Success  bool
	Message  string
	Nickname string
	Kid      int
}

// GiftService 礼品码服务
type GiftService struct {
	repo        storage.Repository
	keyStorage  storage.KeyStorage // 用于 PlayerGiftCode 的存储接口
	captchaPool *captcha.CaptchaPool
	httpClient  *http.Client
	logger      *logrus.Logger
	playerCache sync.Map        // 缓存 PlayerGiftCode 实例
	userCache   *cache.LRUCache // 用户信息缓存 (10分钟TTL)
}

// NewGiftService 创建礼品码服务
func NewGiftService(
	repo storage.Repository,
	keyStorage storage.KeyStorage,
	captchaPool *captcha.CaptchaPool,
	httpClient *http.Client,
	logger *logrus.Logger,
) *GiftService {
	return &GiftService{
		repo:        repo,
		keyStorage:  keyStorage,
		captchaPool: captchaPool,
		httpClient:  httpClient,
		logger:      logger,
		userCache:   cache.NewLRUCache(10 * time.Minute),
	}
}

// RedeemGiftCode 兑换单个礼品码
func (s *GiftService) RedeemGiftCode(ctx context.Context, fid, code string) (*RedeemResult, error) {
	// 添加日志上下文
	log := s.logger.WithFields(logrus.Fields{
		"operation": "redeem_gift_code",
		"fid":       fid,
		"code":      code,
	})

	// 检查是否已兑换
	received, err := s.repo.IsGiftCodeReceived(ctx, fid, code)
	if err != nil {
		log.WithError(err).Error("failed to check if gift code received")
		return nil, errors.NewDatabaseError("check_gift_code_received", err)
	}

	if received {
		log.Info("gift code already received")
		return &RedeemResult{
			FID:     fid,
			Code:    code,
			Success: true,
			Message: "兑换码已兑换",
		}, nil
	}

	// 获取或创建 PlayerGiftCode 实例
	player, err := s.getOrCreatePlayer(ctx, fid)
	if err != nil {
		log.WithError(err).Error("failed to get or create player")
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	// 执行兑换
	result, err := player.GetGift(code)
	if err != nil {
		log.WithError(err).Error("failed to get gift")
		return &RedeemResult{
			FID:     fid,
			Code:    code,
			Success: false,
			Message: fmt.Sprintf("兑换失败: %v", err),
		}, nil
	}

	// 处理兑换结果
	redeemResult := &RedeemResult{
		FID:  fid,
		Code: code,
	}

	if player.Player != nil {
		redeemResult.Nickname = player.Player.Data.Nickname
		redeemResult.Kid = player.Player.Data.Kid
	}

	if result.Code == 0 {
		// 兑换成功
		redeemResult.Success = true
		redeemResult.Message = "兑换成功"

		// 保存兑换记录
		if err := s.repo.SaveGiftCode(ctx, fid, code); err != nil {
			log.WithError(err).Error("failed to save gift code")
			// 不返回错误，因为兑换已成功
		}

		log.Info("gift code redeemed successfully")
	} else {
		// 兑换失败
		redeemResult.Success = false
		redeemResult.Message = result.Msg

		// 特殊情况：已兑换或不存在，也保存记录
		if result.Msg == giftcode.ErrMsgReceived || result.Msg == giftcode.ErrMsgCdkNotFound {
			if err := s.repo.SaveGiftCode(ctx, fid, code); err != nil {
				log.WithError(err).Error("failed to save gift code")
			}
			if result.Msg == giftcode.ErrMsgReceived {
				redeemResult.Success = true
			}
		}

		log.WithField("result_msg", result.Msg).Info("gift code redemption failed")
	}

	return redeemResult, nil
}

// BatchRedeemGiftCode 批量兑换礼品码
// 为所有用户兑换指定的礼品码
// 使用 worker pool 限制并发数
func (s *GiftService) BatchRedeemGiftCode(ctx context.Context, fids []string, code string, workerPoolSize int) ([]*RedeemResult, error) {
	log := s.logger.WithFields(logrus.Fields{
		"operation":        "batch_redeem_gift_code",
		"code":             code,
		"fid_count":        len(fids),
		"worker_pool_size": workerPoolSize,
	})

	log.Info("starting batch redemption")

	// 如果 workerPoolSize 未指定或无效，使用默认值
	if workerPoolSize <= 0 {
		workerPoolSize = 5
	}

	results := make([]*RedeemResult, 0, len(fids))
	var mu sync.Mutex

	// 创建 semaphore channel 限制并发数
	semaphore := make(chan struct{}, workerPoolSize)

	// 使用 WaitGroup 等待所有兑换完成
	var wg sync.WaitGroup

	for _, fid := range fids {
		wg.Add(1)

		// 获取 semaphore
		semaphore <- struct{}{}

		go func(fid string) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放 semaphore

			result, err := s.RedeemGiftCode(ctx, fid, code)
			if err != nil {
				log.WithFields(logrus.Fields{
					"fid":   fid,
					"error": err,
				}).Error("failed to redeem gift code for fid")

				result = &RedeemResult{
					FID:     fid,
					Code:    code,
					Success: false,
					Message: fmt.Sprintf("兑换失败: %v", err),
				}
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(fid)
	}

	wg.Wait()

	log.WithField("result_count", len(results)).Info("batch redemption completed")

	return results, nil
}

// getOrCreatePlayer 获取或创建 PlayerGiftCode 实例
// 使用缓存避免重复初始化
func (s *GiftService) getOrCreatePlayer(ctx context.Context, fid string) (*giftcode.PlayerGiftCode, error) {
	// 尝试从缓存获取
	if cached, ok := s.playerCache.Load(fid); ok {
		return cached.(*giftcode.PlayerGiftCode), nil
	}

	// 创建新实例
	player := giftcode.NewPlayerGiftCode(fid, s.captchaPool.Get, s.keyStorage)

	// 初始化
	if err := player.InitWithContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize player: %w", err)
	}

	// 存入缓存
	s.playerCache.Store(fid, player)

	// 同时缓存用户信息到 LRU 缓存
	if player.Player != nil {
		s.userCache.Set(fid, player.Player)
	}

	return player, nil
}

// GetUserInfo 获取用户信息（带缓存）
func (s *GiftService) GetUserInfo(ctx context.Context, fid string) (*giftcode.DdPlayerMsg, error) {
	// 尝试从缓存获取
	if cached, ok := s.userCache.Get(fid); ok {
		return cached.(*giftcode.DdPlayerMsg), nil
	}

	// 缓存未命中，从数据库或API获取
	player, err := s.getOrCreatePlayer(ctx, fid)
	if err != nil {
		return nil, err
	}

	return player.Player, nil
}
