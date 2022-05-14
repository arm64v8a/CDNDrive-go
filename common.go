package main

import (
	"CDNDrive/drivers"
	"CDNDrive/encoders"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"reflect"
	"strings"
	"time"

	color "CDNDrive/gookit_color"
)

//CDNDrive的格式
type metaJSON struct {
	Time       int64            `json:"time"`
	FileName   string           `json:"filename"`
	Size       int64            `json:"size"`
	Sha1       string           `json:"sha1"`
	BlockDicts []metaJSON_Block `json:"block"`
}

type metaJSON_Block struct {
	URL  string `json:"url"`
	Size int    `json:"size"`
	Sha1 string `json:"sha1"`

	i      int //第几块
	offset int64
}

var _drivers = make([]drivers.Driver, 0)
var _debug bool

func loadDrivers() {
	_drivers = append(_drivers, drivers.NewDriverBilibili())
	_drivers = append(_drivers, drivers.NewDriverBaijia())
	_drivers = append(_drivers, drivers.NewDriverSogou())
	_drivers = append(_drivers, drivers.NewDriverChaoXing())
}

// metaURL -> Driver
func getDriverByMetaLink(metaURL string) drivers.Driver {
	for _, d := range _drivers {
		//Meta2Real成功就说明是对应driver的链接
		if d.Meta2Real(metaURL) != "" {
			return d
		}
	}
	return nil
}

func getDriverByName(name string) drivers.Driver {
	for _, d := range _drivers {
		if name == d.Name() {
			return d
		}
	}
	return nil
}

//resp.Body(photo) -> []byte(file) int=downloadedPhotoSize
func readPhotoBytes(r io.Reader, e encoders.Encoder) ([]byte, error, int) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err, len(b)
	}
	c, d := e.Decode(b)
	return c, d, len(b)
}

//driver过滤，顺便去重
func vaildDrivers(driverString string) map[string]drivers.Driver {
	ds := make(map[string]drivers.Driver)
	for _, name := range strings.Split(driverString, ",") {
		_d := getDriverByName(name)
		if _d != nil {
			ds[name] = _d
		}
	}
	return ds
}

//颜色支持

type colorLogger_t struct {
	prefix func() string
}

var colorLogger *colorLogger_t

func (p *colorLogger_t) Println(a ...interface{}) {
	header := interface{}(p.prefix())
	b := append([]interface{}{header}, a...)
	color.Println(b...)
}

func (p *colorLogger_t) Printf(f string, a ...interface{}) {
	color.Printf(p.prefix()+" "+f, a...)
}

//下面是抄来的

const (
	// B byte
	B = (int64)(1 << (10 * iota))
	// KB kilobyte
	KB
	// MB megabyte
	MB
	// GB gigabyte
	GB
	// TB terabyte
	TB
	// PB petabyte
	PB
)

// ConvertFileSize 文件大小格式化输出
func ConvertFileSize(size int64, precision ...int) string {
	pint := "6"
	if len(precision) == 1 {
		pint = fmt.Sprint(precision[0])
	}
	if size < 0 {
		return "0B"
	}
	if size < KB {
		return fmt.Sprintf("%dB", size)
	}
	if size < MB {
		return fmt.Sprintf("%."+pint+"fKB", float64(size)/float64(KB))
	}
	if size < GB {
		return fmt.Sprintf("%."+pint+"fMB", float64(size)/float64(MB))
	}
	if size < TB {
		return fmt.Sprintf("%."+pint+"fGB", float64(size)/float64(GB))
	}
	if size < PB {
		return fmt.Sprintf("%."+pint+"fTB", float64(size)/float64(TB))
	}
	return fmt.Sprintf("%."+pint+"fPB", float64(size)/float64(PB))
}

// FormatTime 将 Unix 时间戳, 转换为字符串
func FormatTime(t int64) string {
	tt := time.Unix(t, 0).Local()
	year, mon, day := tt.Date()
	hour, min, sec := tt.Clock()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", year, mon, day, hour, min, sec)
}

func randSource(m map[string][]metaJSON_Block) (drivers.Driver, []metaJSON_Block) {
	r := rand.Intn(len(m))
	for k, v := range m {
		if r == 0 {
			return getDriverByName(k), v
		}
		r--
	}
	panic("unreachable")
}

func In(haystack interface{}, needle interface{}) bool {
	sVal := reflect.ValueOf(haystack)
	kind := sVal.Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		for i := 0; i < sVal.Len(); i++ {
			if sVal.Index(i).Interface() == needle {
				return true
			}
		}

		return false
	}

	return false
}
