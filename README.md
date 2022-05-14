# CDNDrive-go

<h4 align="center">☁️ 喵喵喵喵，喵喵喵喵！ ☁️</h4>
<h4 align="center">☁️ 支持任意文件的全速上传与下载 ☁️</h4>

<h4 align="center">冻鳗资源分享(频道) https://t.me/cdndrive</h4>

## 特色

轻量：无复杂依赖，资源占用少

自由：无文件格式与大小限制，无容量限制

稳定：带有分块校验与超时重试机制，在较差的网络环境中依然能确保文件的完整性

快速：支持多线程传输与断点续传，同时借助各个站点的 CDN 资源，能最大化地利用网络环境进行上传与下载

相较于 Python 版 CDNDrive 本项目优势：

1. 内存占用更小
2. Go 语言编写，跨平台便携性好
3. 支持从多个源下载同一个文件，减小链接失效风险
4. 批量下载模式，全集动画一瞬下载
5. 下载支持断点续传
6. 彩色输出

# 下载/运行 说明

Go语言程序, 可直接在[发布页](https://github.com/arm64v8a/CDNDrive-go/releases)下载使用.

如果程序运行时输出乱码, 请检查下终端的编码方式是否为 `UTF-8`.

使用本程序之前, 建议学习一些 linux 基础知识 和 基础命令.

如果未带任何参数运行程序, 程序会提供相关命令的使用说明.

## Windows

程序应在 命令提示符 (Command Prompt) 或 PowerShell 中运行, 在 mintty (例如: GitBash) 可能会有显示问题.

## Linux / macOS

程序应在 终端 (Terminal) 运行.

## Android / iOS

> Android / iOS 移动设备操作比较麻烦, 不建议在移动设备上使用本程序.

安卓, 建议使用 [Termux](https://termux.com) 或 [NeoTerm](https://github.com/NeoTerm/NeoTerm) 或 终端模拟器, 以提供终端环境.

苹果iOS, 需要越狱, 在 Cydia 搜索下载并安装 MobileTerminal, 或者其他提供终端环境的软件.

### Android DNS 问题

由于 Android 系统 不同于正常 Linux 系统的 DNS 机制，预编译的 CDNDrive-go 在 Android 下会出现 read udp connection refuesd 类似错误。

目前推荐解决方法：使用 [Termux](https://termux.com) 安装 Golang 并重新编译本程序。

## 下载文件

```
NAME:
   CDNDrive download - 下载文件

USAGE:
   CDNDrive download [command options] [arguments...]

OPTIONS:
   --https                            强制使用https (default: false)
   --batch                            批量下载模式 (default: false)
   --source-filter value, --sf value  只下载某种链接，如 bdex，用逗号分割
   --replace value, -r value          替换 URL 中某段文字，如 i0.hdslb.com=i1.hdslb.com
   --thread value, -t value           并发连接数 (default: 4)
   --timeout value                    分块传输超时，单位为秒。 (default: 30)
   --help, -h                         show help (default: false)
```

### 使用例：单个文件下载

`CDNDrive download -t 线程数 链接`

单来源下载

`bdex://64ec4075dc3cfa28cf12f147da8f4282d635657b`

多来源下载

`bdex://7fc4accd6cafa0cdd9168cf5ee81a407cabe89a1+sgdrive://100520146/5C0E029D88D39A6FB795AD8D92CBF101+bjdrive://17a65fbb83c249699a9256c3bcd98a6f`

半角加号分割的链接，表示这个文件可以从多个来源下载。

### 使用例：多个文件下载

`CDNDrive download --batch`

然后按照提示操作即可。

## 性能指标

测试时间 20210802 版本 v0.2

下面数字均为本机（普通的笔记本）测试结果

### 上传

1. 图片编码能力

~~TestEncode Speed: 10.73M/s~~

v0.6 [关闭了PNG的压缩](https://github.com/arm64v8a/CDNDrive-go/pull/1)，图片编码速度加快约四倍。

CPU 时间大部分消耗在 png.Encode() 上

由于内存限制，图片使用单线程编码，可能无法完全发挥 CPU 性能。

2. 内存占用

单 driver 16 线程上传，Python CDNDrive 占用约 3GB，CDNDrive-go 占用约 300MB

三 driver 16 线程同时上传，网络顺畅时占用约 400MB ，网络不顺畅时会内存泄露，可以通过调整参数缓解，默认条件下大概在 1GB 左右。
### 下载

默认参数下 (4线程http)

内存占用在 200M 以内，CPU 占用大约一个核心。

一般情况下国内网络均可跑满。

尽量不要使用 https ，因为 https 会使多个下载线程复用同一条连接，容易出现速度瓶颈。

## 支持情况

现在有以下 Driver

| 名称            | 链接         | 上传是否需要登录 | 备注           |
|---------------|------------|----------|--------------|
| BiliBiliDrive | bdex://    | 需要登录     | 陈叔叔家的，推荐！    |
| BaijiaDrive   | bjdrive:// | 无需登录     |              |
| SogouDrive    | sgdrive:// | 无需登录     | 好像服务器会自动清理文件 |
| ChaoXingDrive | cxdrive:// | 需要登录     | 超星学习通的服务     |

欢迎向本项目提交代码添加 Driver （

## 免责声明

+   请自行对重要文件做好本地备份，图床里的图片随时可能被删除。
+   请不要上传含有个人隐私的文件，因为无法删除。
+   尽量不要使用自己帐号上传，以免封号。

## 致谢

原作 [apachecn/CDNDrive](https://github.com/apachecn/CDNDrive)

原作的原作 [Hsury/BiliDrive](https://github.com/Hsury/BiliDrive)

## 猫猫很可爱，请给猫猫打钱

https://nekoquq.github.io/about/

# 部分图床调研

| 名称        | 接口                                               | 接口情况  | 图片存活情况                                                                                                                                                  |
|-----------|--------------------------------------------------|-------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| 今日头条1     | mp.toutiao.com/upload_photo                      | PNG二压 | 未知                                                                                                                                                      |
| 阿里客服1     | kfupload.alibaba.com/mupload                     | 能上传   | 听说会[删除](https://ae01.alicdn.com/kf/U4d394a3b39e24923d834409eb81ef7c0neL.jpg)，有的又[活着](https://ae01.alicdn.com/kf/H5fad35d66dca46108a4898efc0c79f7cT.jpg) |
| 京东1       | myjd.jd.com/afs/common/upload.action             | 貌似失效  | [好像还行](https://img30.360buyimg.com/myjd/jfs/t1/115780/7/160/2878176/5e8821feE7ee8c583/7d151490baadbdc5.png)                                             |
| 腾讯1       | om.qq.com/image/orginalupload                    | PNG二压 | 未知                                                                                                                                                      |
| 百度识图1     | graph.baidu.com/upload                           | 能上传   | 会删除                                                                                                                                                     |
| 小米1       | shopapi.io.mi.com/homemanage/shop/uploadpic      | 貌似失效  | 未知                                                                                                                                                      |
| Telegraph | telegra.ph/upload                                | 墙     | 墙                                                                                                                                                       |
| 掘金1       | cdn-ms.juejin.im/v1/upload                       | 貌似失效  | 未知                                                                                                                                                      |
| 网易1       | you.163.com/xhr/file/upload.json                 | 貌似失效  | 未知                                                                                                                                                      |
| 网易2       | upload.buzz.163.com/picupload                    | 能上传   | [好像还行](https://dingyue.ws.126.net/2020/0707/3fa02122p00qd2k2b00c2d200he00bmg00gb00av.png)                                                               |
| 苏宁1       | review.suning.com/imageload/uploadImg.do         | 要登录   | 未知                                                                                                                                                      |
| 微博1       | picupload.weibo.com/interface/pic_upload.php     | 要登录   | 未知                                                                                                                                                      |
| 360搜索1    | graph.baidu.com/upload                           | 能上传   | 会删除                                                                                                                                                     |
| 悟空问答1     | www.wukong.com/wenda/web/upload/photo/           | 要登录   | 未知                                                                                                                                                      |
| 小米2       | qiye.mi.com/index/upload                         | 要登录   | 未知                                                                                                                                                      |
| 新浪问答1     | iask.sina.com.cn/question/ajax/fileupload        | 要登录   | 未知                                                                                                                                                      |
| 搜狐1       | mp.sohu.com/commons/front/outerUpload/image/file | 要登录   | 未知                                                                                                                                                      |
| CSDN1     | blog-console-api.csdn.net/v1/upload/img          | 要登录   | 未知                                                                                                                                                      |

小网站不收录。

加密文件传输（未测试） https://github.com/Mikubill/transfer
