package captcha

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ocr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr/v20181119"
	"io"
	"strings"
)

type TcCaptchaClient struct {
	client *ocr.Client
}

type aliOcrResponse struct {
	Response struct {
	} `json:"Response"`
}

func NewTcCaptchaClient(id, key string) (*TcCaptchaClient, error) {
	cli, err := createTcClient(id, key)
	if err != nil {
		return nil, err
	}
	return &TcCaptchaClient{
		client: cli,
	}, nil
}

func (t *TcCaptchaClient) DoWithBase64Img(base64Img string) (*CaptchaResponse, error) {
	base64Source := base64Img[strings.Index(base64Img, ",")+1:]
	req := ocr.NewGeneralAccurateOCRRequest()
	req.ImageBase64 = &base64Source
	//req.IsPdf = false
	resp, err := t.client.GeneralAccurateOCR(req)
	if err != nil {
		var tcErr *tcerrors.TencentCloudSDKError
		if errors.As(err, &tcErr) {
			return nil, errors.New(tcErr.Error())
		}
		return nil, err
	}
	if len(resp.Response.TextDetections) == 0 {
		return nil, errors.New("未识别到任何文字")
	}
	return &CaptchaResponse{
		Content: *resp.Response.TextDetections[0].DetectedText,
		Word:    "",
	}, nil
}

func (t *TcCaptchaClient) DoWithReader(r io.Reader) (*CaptchaResponse, error) {
	// 读取转换为base64
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())
	return t.DoWithBase64Img("data:image/png;base64," + base64Img)
}

func createTcClient(id, key string) (*ocr.Client, error) {
	credential := common.NewCredential(
		id, key,
	)
	// 使用临时密钥示例
	// credential := common.NewTokenCredential("SecretId", "SecretKey", "Token")
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ocr.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	return ocr.NewClient(credential, "", cpf)
}
