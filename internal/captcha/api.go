package captcha

import "io"

type CaptchaResponse struct {
	Content string `json:"content"`
	Word    string `json:"word"`
}

type RemoteClient interface {
	DoWithBase64Img(base64Img string) (*CaptchaResponse, error)
	DoWithReader(r io.Reader) (*CaptchaResponse, error)
}
