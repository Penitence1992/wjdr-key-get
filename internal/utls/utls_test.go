package utls

import (
	"net/url"
	"testing"
)

const ddSecretKey = "Uiv#87#SPan.ECsp"

func TestGenSign(t *testing.T) {
	params := url.Values{}
	params.Add("fid", "153928370")
	params.Add("time", "1767599779829")
	s := generateSign(params, ddSecretKey)
	t.Logf("sign: %s", s)
}

func TestWxPush(t *testing.T) {
	PushWithWxPusher("兑换码兑换成功", "兑换码abcd兑换成功", "兑换码abcd全部用户兑换成功, 有用户a,b,c")
}
