package drivers

import (
	"CDNDrive/encoders"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type Driver interface {
	//插件名、显示名
	Name() string
	DisplayName() string

	Headers() map[string]string //http要用
	Encoder() encoders.Encoder  //使用的Encoder

	Exist(hash string) (bool, error)                 //图片是否已存在
	Meta2Real(metaURL string) string                 //bdex:// -> URL （模糊匹配）
	Real2Meta(realURL string) string                 //URL -> bdex://
	Login(username, password string) (string, error) //登录
	CheckCookie(cookie string) (bool, error)
	Upload(data []byte, ctx context.Context, client *http.Client, cookie string) (string, error) //data是图片
}

var Debug bool
var ForceProxy bool
var ProxyPoolURL string

func getProxyTransport() (*http.Transport, error) {
	resp, err := http.Get(ProxyPoolURL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	v2 := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&v2)
	if err != nil {
		return nil, err
	}

	if a, ok := v2["proxy"].(string); ok {
		uri, err := url.Parse("http://" + a)
		if err != nil {
			return nil, err
		}
		return &http.Transport{
			Proxy: http.ProxyURL(uri),
		}, nil
	}
	return nil, errors.New("获取代理失败")
}
