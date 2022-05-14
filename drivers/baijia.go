package drivers

import (
	"CDNDrive/encoders"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
)

type DriverBaijia struct {
	default_url  string
	default_hdrs map[string]string
}

func NewDriverBaijia() *DriverBaijia {
	d := &DriverBaijia{}
	d.default_hdrs = map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36",
	}
	d.default_url = "http://pic.rmb.bdstatic.com/bjh/{md5}.png"
	return d
}

func (d *DriverBaijia) Name() string {
	return "baijia"
}

func (d *DriverBaijia) DisplayName() string {
	return "<cyan>BaijiaDrive</>"
}

func (d *DriverBaijia) Headers() map[string]string {
	return d.default_hdrs
}

func (d *DriverBaijia) Encoder() encoders.Encoder {
	return &encoders.EncoderPNGBMP{}
}

func (d *DriverBaijia) Exist(md5 string) (bool, error) {
	resp, err := http.Head(d.GenURL(md5))
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	}
	return false, nil
}

//Hash->URL
func (d *DriverBaijia) GenURL(md5 string) string {
	return strings.Replace(d.default_url, "{md5}", md5, 1)
}

func (d *DriverBaijia) Meta2Real(metaURL string) string {
	exp, _ := regexp.Compile("bjdrive://([a-fA-F0-9]{32})")
	matchs := exp.FindStringSubmatch(metaURL)
	if len(matchs) < 2 {
		return ""
	}
	return d.GenURL(matchs[1])
}

func (d *DriverBaijia) Real2Meta(realURL string) string {
	exp, _ := regexp.Compile("/bjh/([a-fA-F0-9]{32}).")
	matchs := exp.FindStringSubmatch(realURL)
	if len(matchs) < 2 {
		return ""
	}

	return "bjdrive://" + matchs[1]
}

func (d *DriverBaijia) Login(username, password string) (string, error) {
	return "", errors.New("无需登录")
}

func (d *DriverBaijia) CheckCookie(cookie string) (bool, error) {
	//这个貌似不需要 Cookie 就能传
	return true, nil
}

func (d *DriverBaijia) Upload(data []byte, ctx context.Context, client *http.Client, cookie string) (string, error) {
	//查重
	md5sum := fmt.Sprintf("%x", md5.Sum(data))
	if e, _ := d.Exist(md5sum); e {
		return d.GenURL(md5sum), nil
	}

	//表单上传
	var b bytes.Buffer
	defer b.Reset()
	w := multipart.NewWriter(&b)
	w2, _ := w.CreateFormFile("media", md5sum+".png")
	w2.Write(data)
	w3, _ := w.CreateFormField("type")
	w3.Write([]byte("image"))
	w.Close()

	req, _ := http.NewRequest("POST", "http://rsbjh.baidu.com/builderinner/api/content/file/upload?is_waterlog=0", &b)
	req = req.WithContext(ctx)
	for k, v := range d.Headers() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", w.FormDataContentType())

	//代理
	if ForceProxy && ProxyPoolURL != "" {
		t, err := getProxyTransport()
		if err != nil {
			return "", err
		}
		client.Transport = t
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	//解析
	v := &struct {
		Errno  int    `json:"errno"`
		Errmsg string `json:"errmsg"`
		Ret    struct {
			Url string `json:"https_url"`
		} `json:"ret"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return "", err
	}
	if v.Errno == 0 {
		//校验
		if !strings.Contains(strings.ToLower(v.Ret.Url), md5sum) {
			return "", errors.New("校验值不一致")
		}
		return v.Ret.Url, nil
	} else {
		return "", errors.New(fmt.Sprintf("baidu %d: %s", v.Errno, v.Errmsg))
	}
}
