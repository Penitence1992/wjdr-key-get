package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ocr_api20210707 "github.com/alibabacloud-go/ocr-api-20210707/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/penitence1992/go-server-v1/pkg/errors"
	"io"
	"strings"
)

type AliCaptchaClient struct {
	ocrcli *ocr_api20210707.Client
}

func NewAliCaptchaClient(id, key string) (*AliCaptchaClient, error) {
	cli, err := createAliClient(id, key)
	if err != nil {
		return nil, err
	}
	return &AliCaptchaClient{
		ocrcli: cli,
	}, nil
}

func (c *AliCaptchaClient) DoWithBase64Img(base64Img string) (*CaptchaResponse, error) {
	base64Source := base64Img[strings.Index(base64Img, ",")+1:]
	blob, err := base64.StdEncoding.DecodeString(base64Source)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(blob)
	return c.DoWithReader(buf)
}

func (c *AliCaptchaClient) DoWithReader(r io.Reader) (*CaptchaResponse, error) {
	recognizeHandwritingRequest := &ocr_api20210707.RecognizeHandwritingRequest{
		Body: r,
	}
	runtime := &util.RuntimeOptions{}
	resp, err := c.ocrcli.RecognizeHandwritingWithOptions(recognizeHandwritingRequest, runtime)
	if err != nil {
		return nil, err
	}
	if resp.Body.Code != nil {
		return nil, errors.New(*resp.Body.Code + " " + *resp.Body.Message)
	}
	cresp := &CaptchaResponse{}
	if err = json.Unmarshal([]byte(*resp.Body.Data), &cresp); err != nil {
		return nil, err
	}
	return cresp, nil
}

func createAliClient(id, key string) (_result *ocr_api20210707.Client, _err error) {
	// 工程代码建议使用更安全的无AK方式，凭据配置方式请参见：https://help.aliyun.com/document_detail/378661.html。
	config := new(credentials.Config).SetType("access_key").SetAccessKeyId(id).SetAccessKeySecret(key)
	credential, _err := credentials.NewCredential(config)
	if _err != nil {
		return _result, _err
	}

	opapiConfig := &openapi.Config{
		Credential: credential,
	}
	// Endpoint 请参考 https://api.aliyun.com/product/ocr-api
	opapiConfig.Endpoint = tea.String("ocr-api.cn-hangzhou.aliyuncs.com")
	_result = &ocr_api20210707.Client{}
	_result, _err = ocr_api20210707.NewClient(opapiConfig)
	return _result, _err
}
