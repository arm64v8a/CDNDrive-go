package main

import (
	"CDNDrive/drivers"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

// /tmp/cdndrive-go.conf 保存用户 cookie 信息

type userCookieJson struct {
	Cookie map[string]string
	fp     *os.File
}

func userLogin(driver drivers.Driver, username, password string) error {
	cookie, err := driver.Login(username, password)
	if err != nil {
		return err
	}
	cookieJson := loadUserCookie()
	driverName := driver.Name()
	err = cookieJson.setDriveCookie(driverName, cookie, false)
	if err != nil {
		return err
	}

	return nil
}

func loadUserCookie() *userCookieJson {
	v := &userCookieJson{Cookie: make(map[string]string)}

	//加载不到也没事
	path, _ := os.UserConfigDir()
	f, err := os.OpenFile(filepath.Join(path, "cdndrive-go.conf"), os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		json.NewDecoder(f).Decode(v)
	}
	v.fp = f

	return v
}

func (c *userCookieJson) getCookieByDriverName(name string) string {
	if _debug {
		fmt.Println("Finding cookie for " + name)
	}
	for driver, cookie := range c.Cookie {
		if driver == name {
			if _debug {
				fmt.Println("Cookie is", cookie)
			}
			return cookie
		}
	}
	return ""
}

func (c *userCookieJson) setDriveCookie(name, cookie string, skipCheck bool) (err error) {
	//检查cookie有效性
	if !skipCheck {
		var ok bool
		if ok, err = getDriverByName(name).CheckCookie(cookie); !ok {
			return
		}
	}

	c.Cookie[name] = cookie

	c.fp.Truncate(0)
	c.fp.Seek(0, 0)
	err = json.NewEncoder(c.fp).Encode(c)
	if err != nil {
		return
	}
	err = c.fp.Sync()
	return
}

func HandlerUpload(c *cli.Context, args []string, ds map[string]drivers.Driver) {
	txt_uploadFail := "<fg=black;bg=red>上传失败：</>"

	cookieJson := loadUserCookie()
	blockSize := c.Int("block-size")
	threadN := c.Int("thread")
	blockTimeout := c.Int("timeout")

	//打开文件，读取信息
	f, err := os.OpenFile(args[0], os.O_RDONLY, 0)
	if err != nil {
		colorLogger.Println(txt_uploadFail, err.Error())
		return
	}
	fs, _ := f.Stat()
	fileSize := fs.Size()
	fileName := filepath.Base(f.Name())
	fileName_display := "<yellow>" + fileName + "</>"

	//分块计算
	driverN := len(ds)
	blockN := int(math.Ceil(float64(fileSize) / float64(blockSize)))
	blocks_dict := make([]metaJSON_Block, blockN)
	var _offset int64
	for i, _ := range blocks_dict {
		blocks_dict[i].offset = _offset
		blocks_dict[i].i = i
		if i == blockN-1 { //最后一块
			blocks_dict[i].Size = int(fileSize - int64(_offset))
		} else {
			blocks_dict[i].Size = blockSize
		}
		_offset += int64(blockSize)
	}
	colorLogger.Println("<fg=black;bg=green>正在上传：</><yellow>", f.Name(), "</>大小", ConvertFileSize(fileSize), "分块数", blockN, "分块大小", blockSize, "正在计算 sha1sum")
	if fileSize <= 0 {
		return
	}

	//计算checksum
	hasher := sha1.New()
	f.Seek(0, 0)
	if _, err := io.Copy(hasher, io.NewSectionReader(f, 0, 4*1024*1024)); err != nil {
		colorLogger.Println(txt_uploadFail, "计算 sha1sum 错误：", err.Error())
		return
	}
	sha1_4m := hasher.Sum(nil)
	hasher.Reset()

	f.Seek(0, 0)
	if _, err := io.Copy(hasher, f); err != nil {
		colorLogger.Println(txt_uploadFail, "计算 sha1sum 错误：", err.Error())
		return
	}
	sha1_all := hasher.Sum(nil)
	f.Seek(0, 0)

	sha1sum := fmt.Sprintf("%x", sha1_all)
	sha1_4m = sha1_4m //TODO save history

	//发送任务
	lock := &sync.Mutex{} //TODO 这是什么锁
	ctx, cancel := context.WithCancel(context.Background())

	chanTasks := make([]chan *metaJSON_Block, driverN)
	chanStatus := make([]chan int, driverN)
	finishMaps := make([][]bool, driverN)
	finishurls := make([][]string, driverN)
	ctxs := make([]context.Context, driverN)
	cancels := make([]context.CancelFunc, driverN)

	cache := &worker_up_cache{
		lock_encode:               &sync.Mutex{},
		photoCacheProducer:        make([]int, blockN),
		photoCacheFinishedCounter: make([]int, blockN),
		photoCacheWaiter:          make([]*sync.WaitGroup, blockN),
		photoCache:                make([][]byte, blockN),
		driverN:                   driverN,
		cacheSize:                 c.Int("cache-size"),
	}

	for i, _ := range chanTasks {
		chanTasks[i] = make(chan *metaJSON_Block, blockN)
		chanStatus[i] = make(chan int)
		finishMaps[i] = make([]bool, blockN)
		finishurls[i] = make([]string, blockN)
		ctxs[i], cancels[i] = context.WithCancel(ctx)

		for j, _ := range blocks_dict {
			chanTasks[i] <- &blocks_dict[j]
		}
	}

	//上面的工作只用做一次，下面开始对各个driver上传
	var ii = 0
	wg := &sync.WaitGroup{}
	wg.Add(driverN)
	for _, _d := range ds {
		cookie := cookieJson.getCookieByDriverName(_d.Name())

		//上传进度控制
		go func(d drivers.Driver, ii int) {
			var finishedBlockCounter int
			time_start := time.Now()
			defer wg.Done()
			for {
				select {
				case <-ctxs[ii].Done():
					return
				case finishedBlockID := <-chanStatus[ii]:
					if finishedBlockID < 0 { //负数是出错代码，此时该driver退出
						colorLogger.Println(txt_uploadFail, d.DisplayName(), fileName_display)
						cache.driverN-- //免得又内存泄露，，，
						cancels[ii]()
						return
					}

					finishMaps[ii][finishedBlockID] = true
					finishedBlockCounter++

					if finishedBlockCounter == blockN {
						colorLogger.Println(d.DisplayName(), "上传完成，开始编码并上传索引图片。")

						//这个是要上传的meta
						blocks_dict_copy := make([]metaJSON_Block, blockN)
						for i, _ := range blocks_dict_copy {
							blocks_dict_copy[i].i = i
							blocks_dict_copy[i].Sha1 = blocks_dict[i].Sha1
							blocks_dict_copy[i].Size = blocks_dict[i].Size
							blocks_dict_copy[i].URL = finishurls[ii][i]
						}
						v := &metaJSON{
							Time:       time.Now().Unix(),
							FileName:   fileName,
							Size:       fileSize,
							Sha1:       sha1sum,
							BlockDicts: blocks_dict_copy,
						}
						data, _ := json.Marshal(v)

						try_max := 10
						for i := 0; i < try_max; i++ { //尝试10次
							url, err := d.Upload(d.Encoder().Encode(data), ctx, http.DefaultClient, cookie)

							if err != nil {
								if i < try_max-1 {
									colorLogger.Println(d.DisplayName(), "索引图片第", i+1, "次上传失败，重试。")
								} else {
									colorLogger.Println(d.DisplayName(), "索引图片第", i+1, "次下载失败，不重试，文件上传失败。")
									go func() { chanStatus[ii] <- -2 }()
									break
								}
							} else {
								seconds := time.Now().Sub(time_start).Seconds()
								colorLogger.Println(d.DisplayName(), fileName_display, "上传完毕，用时", seconds, "秒，平均速度", ConvertFileSize(int64(float64(fileSize)/seconds)))
								colorLogger.Println(d.DisplayName(), fileName_display, "上传完毕 <green>META URL</> ->", d.Real2Meta(url))
								cancels[ii]()
								return
							}
						}
					}
				}
			}
		}(_d, ii)

		for j := 0; j < threadN; j++ {
			up := &worker_up{
				workerID:     j,
				cache:        cache,
				blockTimeout: blockTimeout,
			}

			go up.up(chanTasks[ii], chanStatus[ii], ctxs[ii], cookie, _d, f, lock, finishMaps[ii], finishurls[ii])
		}

		ii++
	}

	wg.Wait()
	cancel()
}

//缓存图片
type worker_up_cache struct {
	lock_encode               *sync.Mutex
	photoCache                [][]byte
	photoCacheProducer        []int
	photoCacheFinishedCounter []int
	photoCacheWaiter          []*sync.WaitGroup
	driverN                   int
	cacheCounter              int
	cacheSize                 int
}

//TODO 迁移参数
type worker_up struct {
	workerID     int
	blockTimeout int

	cache *worker_up_cache
}

func (p *worker_up) up(chanTask chan *metaJSON_Block, chanStatus chan int, ctx context.Context, cookie string, d drivers.Driver, f *os.File, lock *sync.Mutex, finishMap []bool, finishurls []string) {
	client := &http.Client{}
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-chanTask:
			try_max := 10
			for i := 0; i < try_max; i++ { //尝试10次
				err := func() (err error) {
					var photo []byte
					var wait bool

					lock.Lock()
					if p.cache.photoCacheProducer[task.i] == 0 {
						wait = true
						p.cache.photoCacheProducer[task.i] = 1 //在做了在做了

						p.cache.photoCacheWaiter[task.i] = &sync.WaitGroup{}
						p.cache.photoCacheWaiter[task.i].Add(1)

						go func() {
							p.cache.lock_encode.Lock()
							defer p.cache.lock_encode.Unlock()
							p.cache.cacheCounter++
							for p.cache.cacheCounter >= p.cache.cacheSize { //手动限制内存，，，
								colorLogger.Println("<red>缓存过多，进入等待模式</>")
								time.Sleep(time.Second * 10)
							}

							//读取文件内容
							f.Seek(task.offset, 0)
							data := make([]byte, task.Size)
							_, err = f.Read(data)
							if err != nil {
								return
							}

							//sha1sum 只计算一次
							var sha1sum string
							if task.Sha1 == "" {
								sha1sum = fmt.Sprintf("%x", sha1.Sum(data))
								task.Sha1 = sha1sum
							} else {
								sha1sum = task.Sha1
							}

							//这个encode特别耗CPU和内存
							//TODO 这样变成只用一个CPU编码了...
							p.cache.photoCache[task.i] = d.Encoder().Encode(data)

							p.cache.photoCacheWaiter[task.i].Done()
							p.cache.photoCacheProducer[task.i] = 2 //好了

							if _debug {
								colorLogger.Println("<cyan>Encoded:", task.i, "</>")
							}
						}() //防卡
					} else if p.cache.photoCacheProducer[task.i] == 1 {
						wait = true
					}
					lock.Unlock()

					//获编码好的图片
					if wait { //防卡
						p.cache.photoCacheWaiter[task.i].Wait()
					}
					photo = p.cache.photoCache[task.i]

					//防卡？
					ctx2, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*time.Duration(p.blockTimeout)))

					//上传，这里task共用所以不能设置url
					finishurls[task.i], err = d.Upload(photo, ctx2, client, cookie)

					//清理缓存
					if err == nil || i == try_max-1 {
						p.cache.photoCacheFinishedCounter[task.i]++
						if p.cache.photoCacheFinishedCounter[task.i] >= p.cache.driverN {
							p.cache.photoCache[task.i] = nil
							p.cache.cacheCounter--
							if _debug {
								colorLogger.Println("<green>free:", task.i, "</>")
								//TODO 是不是还有内存泄露？
							}
						}
					}

					cancel()
					return
				}()
				if err != nil {
					colorLogger.Println(d.DisplayName(), "分块", task.i+1, "<red>错误代码：</>", err.Error())
					if i < try_max-1 {
						colorLogger.Println(d.DisplayName(), "分块", task.i+1, "第", i+1, "次上传失败，重试。")
					} else {
						colorLogger.Println(d.DisplayName(), "分块", task.i+1, "第", i+1, "次上传失败，不重试，文件上传失败。")
						chanStatus <- -1 //停止代码 -1 上传失败
					}
				} else {
					chanStatus <- task.i
					colorLogger.Println(d.DisplayName(), "分块", task.i+1, "/", len(finishurls), "上传成功。")
					break
				}
			}
		}
	}
}
