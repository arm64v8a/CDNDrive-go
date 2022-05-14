package main

import (
	"CDNDrive/drivers"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	//初始化
	loadDrivers()
	colorLogger = &colorLogger_t{
		prefix: func() string { return "[" + FormatTime(time.Now().Unix()) + "]" },
	}

	flag_drivers := &cli.StringFlag{
		Name:    "driver",
		Aliases: []string{"d"},
		Usage: "上传 driver，同时上传多个地方请用逗号分割，必须为以下的一个或多个： " + func() (txt string) {
			for i := 0; i < len(_drivers); i++ {
				txt += _drivers[i].Name()
				if i != len(_drivers)-1 {
					txt += ", "
				}
			}
			return
		}(),
	}
	flag_threadN := &cli.IntFlag{
		Name:    "thread",
		Aliases: []string{"t"},
		Usage:   "并发连接数",
		Value:   4,
	}
	flag_blockTimeout := &cli.IntFlag{
		Name:  "timeout",
		Usage: "分块传输超时，单位为秒。",
		Value: 30,
	}

	app := &cli.App{
		Name:    "CDNDrive-go",
		Usage:   "Make Picbeds Great Cloud Storages!",
		Version: "v0.9.1",
		Authors: []*cli.Author{
			&cli.Author{
				Name: "猫村あおい",
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "debug",
			}},
		Before: func(c *cli.Context) error {
			_debug = c.Bool("debug")
			drivers.Debug = _debug
			return nil
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "download",
				Aliases: []string{"d"},
				Usage:   "下载文件",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "https",
						Usage: "强制使用https",
					}, &cli.BoolFlag{
						Name:  "batch",
						Usage: "批量下载模式",
					}, &cli.StringFlag{
						Name:    "source-filter",
						Aliases: []string{"sf"},
						Usage:   "只下载某种链接，如 bdex，用逗号分割",
					}, &cli.StringSliceFlag{
						Name:    "replace",
						Aliases: []string{"r"},
						Usage:   "替换 URL 中某段文字，如 i0.hdslb.com=i1.hdslb.com",
					},
					flag_threadN,
					flag_blockTimeout,
				},
				Action: func(c *cli.Context) error {
					if c.NArg() == 0 && !c.Bool("batch") {
						cli.ShowCommandHelpAndExit(c, "download", 1)
					}
					HandlerDownload(c, c.Args().Slice())
					return nil
				},
			}, &cli.Command{
				Name:    "upload",
				Aliases: []string{"u"},
				Usage:   "上传文件",
				Flags: []cli.Flag{
					flag_threadN,
					flag_drivers,
					flag_blockTimeout,
					&cli.IntFlag{
						Name:    "block-size",
						Aliases: []string{"b"},
						Usage:   "分块大小，单位为字节",
						Value:   4 * 1024 * 1024,
					}, &cli.StringFlag{
						Name:    "proxy-pool",
						Aliases: []string{"pp"},
						Usage:   "代理池",
					}, &cli.IntFlag{
						Name:    "proxy-time",
						Aliases: []string{"pt"},
						Usage:   "遇到 bilibili -412 时，启用代理的时间，单位为分钟。",
						Value:   10,
					}, &cli.BoolFlag{
						Name:    "proxy-force",
						Aliases: []string{"pf"},
						Usage:   "强制启用代理池",
					}, &cli.IntFlag{
						Name:    "cache-size",
						Aliases: []string{"cs"},
						Usage:   "最大缓存分块数，仅对多 driver 上传有效，内存不足时建议调低。",
						Value:   40,
					},
				},
				Action: func(c *cli.Context) error {
					ds := vaildDrivers(c.String("driver"))
					if c.NArg() == 0 || len(ds) == 0 {
						cli.ShowCommandHelpAndExit(c, "upload", 1)
					}

					if _debug { //内存分析
						go func() {
							http.ListenAndServe("127.0.0.1:9459", nil)
						}()
					}

					drivers.ForceProxy = c.Bool("proxy-force")
					getDriverByName("bili").(*drivers.DriverBilibili).SetProxyPool(c.String("proxy-pool"), c.Int("proxy-time"))
					HandlerUpload(c, c.Args().Slice(), ds)
					return nil
				},
			}, &cli.Command{
				Name:    "cookie",
				Aliases: []string{"c"},
				Usage:   "有些图床上传需要登录，请提供小饼干。",
				Flags: []cli.Flag{
					flag_drivers,
					&cli.BoolFlag{
						Name:  "force",
						Usage: "强制设置，跳过 cookie 有效性检查",
					},
				},
				Action: func(c *cli.Context) error {
					ds := vaildDrivers(c.String("driver"))
					if c.NArg() == 0 || len(ds) == 0 {
						cli.ShowCommandHelpAndExit(c, "cookie", 1)
					}

					if len(ds) > 1 {
						fmt.Println("设置 cookie 时一次只能输入一个 driver")
						return nil
					}

					cookieJson := loadUserCookie()
					for name, _ := range ds {
						err := cookieJson.setDriveCookie(name, c.Args().Get(0), c.Bool("force"))
						if err == nil {
							fmt.Println("设置成功")
						} else {
							fmt.Println("设置失败", err.Error())
						}
					}
					return nil
				},
			}, &cli.Command{
				Name:    "login",
				Aliases: []string{"l"},
				Usage:   "使用用户名和密码登录",
				Flags: []cli.Flag{
					flag_drivers,
					&cli.StringFlag{
						Name:    "username",
						Aliases: []string{"u"},
						Usage:   "用户名",
					},
					&cli.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "密码",
					},
				},
				Action: func(c *cli.Context) error {
					ds := vaildDrivers(c.String("driver"))
					if len(c.String("username")) == 0 || len(c.String("password")) == 0 || len(ds) == 0 {
						cli.ShowCommandHelpAndExit(c, "login", 1)
					}

					if len(ds) > 1 {
						fmt.Println("登录时一次只能输入一个 driver")
						return nil
					}

					for _, driver := range ds {
						err := userLogin(driver, c.String("username"), c.String("password"))
						if err == nil {
							fmt.Println("登录成功")
						} else {
							fmt.Println("登录失败", err.Error())
						}
					}

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
