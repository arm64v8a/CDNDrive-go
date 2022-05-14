package drivers

import (
	"CDNDrive/encoders"
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
)

//TODO 整天清理文件，爬

type DriverSogou struct {
	default_url  string
	default_hdrs map[string]string
}

func NewDriverSogou() *DriverSogou {
	d := &DriverSogou{}
	d.default_hdrs = map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Encoding": " gzip, deflate",
	}
	d.default_url = "http://img01.sogoucdn.com/app/a/{hash}"
	return d
}

func (d *DriverSogou) Name() string {
	return "sogou"
}

func (d *DriverSogou) DisplayName() string {
	return "<cyan>SogouDrive</>"
}

func (d *DriverSogou) Headers() map[string]string {
	return d.default_hdrs
}

func (d *DriverSogou) Encoder() encoders.Encoder {
	return &encoders.EncoderPNGBMP{}
}

func (d *DriverSogou) Exist(hash string) (bool, error) {
	resp, err := http.Head(d.GenURL(hash))
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	}
	return false, nil
}

//Hash->URL
func (d *DriverSogou) GenURL(hash string) string {
	return strings.Replace(d.default_url, "{hash}", hash, 1)
}

func (d *DriverSogou) Meta2Real(metaURL string) string {
	exp, _ := regexp.Compile("sgdrive://([0-9]+/[a-fA-F0-9]{32})")
	matchs := exp.FindStringSubmatch(metaURL)
	if len(matchs) < 2 {
		return ""
	}
	return d.GenURL(matchs[1])
}

func (d *DriverSogou) Real2Meta(realURL string) string {
	exp, _ := regexp.Compile("/app/a/([0-9]+/[a-fA-F0-9]{32})")
	matchs := exp.FindStringSubmatch(realURL)
	if len(matchs) < 2 {
		return ""
	}

	return "sgdrive://" + matchs[1]
}

func (d *DriverSogou) Login(username, password string) (string, error) {
	return "", errors.New("无需登录")
}

func (d *DriverSogou) CheckCookie(cookie string) (bool, error) {
	//这个貌似不需要 Cookie 就能传
	return true, nil
}

func (d *DriverSogou) Upload(data []byte, ctx context.Context, client *http.Client, cookie string) (string, error) {
	//查重
	md5sum := fmt.Sprintf("%x", md5.Sum(data))
	hash := fmt.Sprintf("100520146/%s", strings.ToUpper(md5sum))
	if e, _ := d.Exist(hash); e {
		return d.GenURL(hash), nil
	}

	//表单上传
	var b bytes.Buffer
	defer b.Reset()
	w := multipart.NewWriter(&b)

	//就你这么麻烦
	h := make(textproto.MIMEHeader)
	escapeQuotes := func(s string) string {
		return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
	}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		escapeQuotes("file"), escapeQuotes(md5sum+".png")))
	h.Set("Content-Type", "image/png")
	w2, _ := w.CreatePart(h)
	w2.Write(data)
	w.Close()

	req, _ := http.NewRequest("POST", "http://pic.sogou.com/ris_upload", &b)
	req = req.WithContext(ctx)
	for k, v := range d.Headers() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", w.FormDataContentType())

	// fuck CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil && err != http.ErrUseLastResponse {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	//解析
	location := resp.Header.Get("Location")
	exp, _ := regexp.Compile("query=(http.*img.*sogoucdn.*app[^&]+)")
	matchs := exp.FindStringSubmatch(location)
	if len(matchs) < 2 {
		return "", errors.New("未返回图片地址")
	}
	location, err = url.QueryUnescape(matchs[1])

	//校验
	if !strings.Contains(strings.ToLower(location), md5sum) {
		return "", errors.New("校验值不一致")
	}
	return location, err
}
