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

type DriverChaoXing struct {
	default_url  string
	default_hdrs map[string]string
}

func (d *DriverChaoXing) Exist(hash string) (bool, error) {
	return false, nil
}

func NewDriverChaoXing() *DriverChaoXing {
	d := &DriverChaoXing{}
	d.default_hdrs = map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36",
	}
	d.default_url = "http://p.ananas.chaoxing.com/star3/origin/{obj_id}.png"
	return d
}

func (d *DriverChaoXing) Name() string {
	return "chaoxing"
}

func (d *DriverChaoXing) DisplayName() string {
	return "<cyan>ChaoXingDrive</>"
}

func (d *DriverChaoXing) Headers() map[string]string {
	return d.default_hdrs
}

func (d *DriverChaoXing) Encoder() encoders.Encoder {
	return &encoders.EncoderPNGBMP{}
}

//Hash->URL
func (d *DriverChaoXing) GenURL(md5 string) string {
	return strings.Replace(d.default_url, "{obj_id}", md5, 1)
}

func (d *DriverChaoXing) Meta2Real(metaURL string) string {
	exp, _ := regexp.Compile("cxdrive://([a-fA-F0-9]{32})")
	matchs := exp.FindStringSubmatch(metaURL)
	if len(matchs) < 2 {
		return ""
	}
	return d.GenURL(matchs[1])
}

func (d *DriverChaoXing) Real2Meta(realURL string) string {
	exp, _ := regexp.Compile("([a-fA-F0-9]{32}).")
	matchs := exp.FindStringSubmatch(realURL)
	if len(matchs) < 2 {
		return ""
	}

	return "cxdrive://" + matchs[1]
}

func (d *DriverChaoXing) CheckCookie(cookie string) (bool, error) {
	req, _ := http.NewRequest("GET", "http://mooc1-1.chaoxing.com/api/workTestPendingNew", nil)
	for k, v := range d.Headers() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Cookie", cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ReadAllDecompress(resp)
	if !(strings.Index(string(data), "登录") > 0) {
		return true, nil
	}
	return false, errors.New("chaoxing: " + string(data))
}

func (d *DriverChaoXing) Upload(data []byte, ctx context.Context, client *http.Client, cookie string) (string, error) {
	md5sum := fmt.Sprintf("%x", md5.Sum(data))

	//表单上传
	var b bytes.Buffer
	defer b.Reset()
	w := multipart.NewWriter(&b)
	w2, _ := w.CreateFormFile("attrFile", md5sum+".png")
	w2.Write(data)
	w.Close()

	req, _ := http.NewRequest("POST", "http://notice.chaoxing.com/pc/files/uploadNoticeFile", &b)
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
		AttFile struct {
			AttClouddisk struct {
				Size       int    `json:"size"`
				FileSize   string `json:"fileSize"`
				Isfile     bool   `json:"isfile"`
				Modtime    int64  `json:"modtime"`
				ParentPath string `json:"parentPath"`
				Icon       string `json:"icon"`
				Name       string `json:"name"`
				DownPath   string `json:"downPath"`
				ShareURL   string `json:"shareUrl"`
				Suffix     string `json:"suffix"`
				FileID     string `json:"fileId"`
			} `json:"att_clouddisk"`
			AttachmentType int `json:"attachmentType"`
		} `json:"att_file"`
		Type   string `json:"type"`
		URL    string `json:"url"`
		Status bool   `json:"status"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return "", err
	}
	if v.Status == true {
		//return d.GenURL(v.AttFile.AttClouddisk.FileID), nil
		return strings.Split(v.URL, "?")[0], nil
	} else {
		return "", errors.New(fmt.Sprintf("chaoxing error"))
	}
}
