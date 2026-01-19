package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	username = "15279244633"
	password = "q1w2e3r4t5"
	sid      = 7
)

var key = "bc0520 bc520 vip111 vip222 vip333 vip555 vip666 vip777 vip888 vip999 vip1000 vip1215 vip2000 vip2025 vip0105  vip0112  vip0119  vip0127   vip0210  vip0212  vip0223  vip0304  vip0308  vip0318   vip0331 vip0412 bc0415 FL0501 vip0511 vip520 FL0531  vip0615   zqwn0618  bc071  bc0710   vip0718   vip0810"
var okeys = "vip111 vip222 vip333 vip666 vip777 vip888 vip999 vip1000 vip1215 vip1222 vip0105 vip0112 vip0119 vip0127 vip0210 vip0212 vip0223 vip0304 vip2000 vip2025 vip0331 vip0218 FL0501 vip0615 zqwn0618"
var newsKeys = []string{
	"vip0818", "vip0707",
}
var keepmap = map[string]struct{}{}

type loginResp struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

func main() {
	keepmap = make(map[string]struct{})
	//keys := strings.Split(key, " ")
	//keys = append(keys, strings.Split(okeys, " ")...)
	keys := newsKeys
	url := "http://sdk.gaz.tw:96/iapi/?do=login&account=%s&password=%s"
	getKey := "http://sdk.gaz.tw:96/iapi/?do=getCDKGifts&sid=7&cdk=%s"
	fullurl := fmt.Sprintf(url, username, password)
	httpClient := &http.Client{}
	resp, err := httpClient.Get(fullurl)
	panicHas(err)
	defer resp.Body.Close()
	blob, err := io.ReadAll(resp.Body)
	panicHas(err)
	logResp := &loginResp{}
	panicHas(json.Unmarshal(blob, logResp))
	token := logResp.Data.Token
	for _, k := range keys {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if _, ok := keepmap[key]; ok {
			continue
		}
		keepmap[key] = struct{}{}
		req, _ := http.NewRequest("GET", fmt.Sprintf(getKey, key), nil)
		req.Header.Set("Authorization", token)
		req.Header.Set("referer", fmt.Sprintf("http://sdk.gaz.tw:96/desk/?accessToken=%s", token))
		req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0")
		getResp, _ := httpClient.Do(req)
		blob, _ := io.ReadAll(getResp.Body)
		fmt.Printf("获取key: %s", key)
		fmt.Println(string(blob))
		_ = getResp.Body.Close()
	}
}

func panicHas(err error) {
	if err != nil {
		panic(err)
	}
}
