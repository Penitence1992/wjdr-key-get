package job

import (
	"cdk-get/internal/captcha"
	"cdk-get/internal/config"
	"cdk-get/internal/giftcode"
	"cdk-get/internal/svc"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var fidsDefault = []string{
	"366184723",
}

type GetCodeJob struct {
	svcCtx    *svc.ServiceContext
	cliKeep   map[string]*giftcode.PlayerGiftCode
	clients   []captcha.RemoteClient
	shardLock sync.Mutex
	idx       int
}

func NewGetCodeJob(svcCtx *svc.ServiceContext) *GetCodeJob {
	clients, err := initClients()
	if err != nil {
		panic(err)
	}
	return &GetCodeJob{
		svcCtx:    svcCtx,
		cliKeep:   make(map[string]*giftcode.PlayerGiftCode),
		clients:   clients,
		shardLock: sync.Mutex{},
	}
}

func (g *GetCodeJob) Run(ctx context.Context) {
	codes, err := g.svcCtx.SqlClient.GetTask()
	if len(codes) == 0 {
		logrus.Infof("未发现代办任务")
	}
	if err != nil {
		logrus.Errorf("获取代办任务失败: %v", err)
		return
	}
	fids, err := g.svcCtx.SqlClient.GetFids()
	if err != nil {
		logrus.Errorf("获取处理人失败: %v", err)
		return
	}
	if len(fids) == 0 {
		fids = fidsDefault
	}
	for _, code := range codes {
		g.processCodeSafely(code, fids)
	}
}

func (g *GetCodeJob) processCodeSafely(code string, fids []string) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("处理code %s 时发生panic: %v", code, err)
			logrus.Errorf("堆栈信息: %s", debug.Stack())

			// Update retry count and error on panic
			ctx := context.Background()
			task, _ := g.svcCtx.Repository.GetTaskByCode(ctx, code)
			if task != nil {
				retryCount := task.RetryCount + 1
				errorMsg := fmt.Sprintf("Panic: %v", err)
				_ = g.svcCtx.Repository.UpdateTaskRetry(ctx, code, retryCount, errorMsg)
			}
		}
	}()

	logrus.Infof("开始执行code: %s任务, 处理人: %v", code, fids)
	startTime := time.Now()

	ctx := context.Background()
	alldone, msg, err := g.once(ctx, code, fids)

	if err != nil {
		logrus.Errorf("GetCodeJob GetTask err: %v", err)

		// Update retry count and error on failure
		task, _ := g.svcCtx.Repository.GetTaskByCode(ctx, code)
		if task != nil {
			retryCount := task.RetryCount + 1
			_ = g.svcCtx.Repository.UpdateTaskRetry(ctx, code, retryCount, err.Error())
		}
	} else if alldone {
		// Mark task as complete with timestamp
		completedAt := time.Now()
		if err := g.svcCtx.Repository.UpdateTaskComplete(ctx, code, completedAt); err != nil {
			logrus.Errorf("GetCodeJob UpdateTaskComplete err: %v", err)
		} else {
			// Send notification using NotificationService
			if g.svcCtx.NotificationService != nil {
				title := "兑换码兑换成功"
				summary := fmt.Sprintf("兑换码[%s]兑换成功", code)
				_ = g.svcCtx.NotificationService.SendAndSave(ctx, title, summary, msg)
			}
		}
	}
	logrus.Infof("任务执行信息: %s", msg)
	endTime := time.Now()
	logrus.Infof("完成code: %s任务, 耗时: %s", code, endTime.Sub(startTime).String())
}

func (g *GetCodeJob) once(ctx context.Context, code string, fids []string) (bool, string, error) {
	repository := g.svcCtx.SqlClient
	cliKeep := g.cliKeep
	if code == "" {
		return false, "", errors.New("code is empty")
	}
	var (
		notFound bool
		alldone  = true
		line     = strings.Builder{}
	)
	for _, fid := range fids {
		var (
			gfc *giftcode.PlayerGiftCode
			ok  bool
			msg string
		)
		if gfc, ok = cliKeep[fid]; !ok {
			gfc = giftcode.NewPlayerGiftCode(fid, g.getClient, repository)
			if err := gfc.Init(); err != nil {
				return false, "", err
			}
			cliKeep[fid] = gfc
		}

		line.WriteString(fmt.Sprintf("fid:%v, 昵称:%v, 区服:%v", gfc.Player.Data.Fid, gfc.Player.Data.Nickname, gfc.Player.Data.Kid))
		if ok, notFound, msg = g.getOnceCodeWithOneFid(ctx, code, gfc); !ok {
			alldone = false
		}
		if notFound {
			break
		}
		line.WriteString(fmt.Sprintf(" 结果: %s \n", msg))
	}
	if notFound {
		for _, fid := range fids {
			_ = repository.Save(fid, code)
		}
		return true, fmt.Sprintf("兑换码:%s 不存在", code), nil
	}
	return alldone, line.String(), nil
}

func (g *GetCodeJob) getOnceCodeWithOneFid(ctx context.Context, code string, gfc *giftcode.PlayerGiftCode) (done bool, notFound bool, msg string) {
	repository := g.svcCtx.SqlClient
	if exists, err := repository.IsReceived(gfc.Fid, code); err != nil {
		done = false
		msg = fmt.Sprintf("%v", err)
		return
	} else if exists {
		done = true
		msg = "兑换码已兑换"
		return
	}

	// 获取兑换码
	result, err := gfc.GetGift(code)
	if err != nil {
		done = false
		msg = fmt.Sprintf("%v", err)
		return
	}
	// 处理异常
	if result.Code != 0 {
		if result.Msg == giftcode.ErrMsgReceived {
			done = true
			msg = "兑换码已兑换"
			// 如果已经接受过, 则保存
			_ = repository.Save(gfc.Fid, code)
		} else if result.Msg == giftcode.ErrMsgCdkNotFound {
			done = true
			msg = "CDK不存在"
			notFound = true
			_ = repository.Save(gfc.Fid, code)
		} else {
			done = false
			msg = result.Msg
			// 写入失败次数
			task, _ := g.svcCtx.Repository.GetTaskByCode(ctx, code)
			if task != nil {
				retryCount := task.RetryCount + 1
				_ = g.svcCtx.Repository.UpdateTaskRetry(ctx, code, retryCount, result.Msg)
			}
		}
		return
	} else {
		done = true
		msg = "兑换成功"
		_ = repository.Save(gfc.Fid, code)
		return
	}
}

func (g *GetCodeJob) DelayTime() time.Duration {
	return 2 * time.Second
}

func (g *GetCodeJob) PeriodTime() time.Duration {
	return 30 * time.Second
}

func (g *GetCodeJob) Name() string {
	return "GetCodeJob"
}

func (g *GetCodeJob) getClient() captcha.RemoteClient {
	if len(g.clients) == 0 {
		return nil
	}
	g.shardLock.Lock()
	cli := g.clients[g.idx%len(g.clients)]
	g.idx++
	g.shardLock.Unlock()
	// 重置为0
	if g.idx > 10000 {
		g.idx = 0
	}
	return cli
}

func initClients() (clients []captcha.RemoteClient, err error) {
	// 尝试从配置文件加载配置
	cfg, err := config.LoadConfig("./etc/config.yaml")
	if err != nil {
		logrus.Warnf("加载配置文件失败，使用环境变量: %v", err)
		// 如果配置文件加载失败，使用默认配置（会从环境变量覆盖）
		cfg, _ = config.LoadConfig("")
	}

	// 遍历所有配置的提供商
	for _, provider := range cfg.Captcha.Providers {
		switch provider.Type {
		case "ali":
			if provider.AccessKey != "" && provider.SecretKey != "" {
				alicli, err := captcha.NewAliCaptchaClient(provider.AccessKey, provider.SecretKey)
				if err == nil {
					clients = append(clients, alicli)
					logrus.Infof("成功初始化阿里云OCR客户端")
				} else {
					logrus.Errorf("初始化阿里云图片识别错误: %v", err)
				}
			}
		case "tencent":
			if provider.AccessKey != "" && provider.SecretKey != "" {
				tccli, err := captcha.NewTcCaptchaClient(provider.AccessKey, provider.SecretKey)
				if err == nil {
					clients = append(clients, tccli)
					logrus.Infof("成功初始化腾讯云OCR客户端")
				} else {
					logrus.Errorf("初始化腾讯云图片识别错误: %v", err)
				}
			}
		case "google":
			if provider.CredentialsJSON != "" {
				googlecli, err := captcha.NewGoogleCaptchaClient(provider.CredentialsJSON)
				if err == nil {
					clients = append(clients, googlecli)
					logrus.Infof("成功初始化Google Vision OCR客户端")
				} else {
					logrus.Errorf("初始化Google Vision图片识别错误: %v", err)
				}
			}
		default:
			logrus.Warnf("未知的验证码提供商类型: %s", provider.Type)
		}
	}

	// 如果没有配置任何提供商，尝试使用环境变量（向后兼容）
	if len(clients) == 0 {
		logrus.Warn("未找到配置的OCR提供商，尝试使用环境变量")
		if accessKey := os.Getenv("ACCESS_KEY"); accessKey != "" {
			if secretKey := os.Getenv("ACCESS_SECRET"); secretKey != "" {
				alicli, err := captcha.NewAliCaptchaClient(accessKey, secretKey)
				if err == nil {
					clients = append(clients, alicli)
					logrus.Infof("从环境变量成功初始化阿里云OCR客户端")
				} else {
					logrus.Errorf("从环境变量初始化阿里云图片识别错误: %v", err)
				}
			}
		}
	}

	if len(clients) == 0 {
		return nil, errors.New("未能初始化任何OCR客户端")
	}

	logrus.Infof("共初始化 %d 个OCR客户端", len(clients))
	return clients, nil
}
