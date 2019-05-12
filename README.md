# K

K 是个 CLI 工具，提供简便的杀死进程的方法。K 基本是 [fkill-cli](https://github.com/sindresorhus/fkill-cli) 的 golang 版本，但不仅支持按进程名称，而且支持按 Pid、进程所监听的本地的端口来过滤。支持 Linux 和 macOS 系统。

<img src="./show.jpg" alt="" style="with:400px;height:202px">

## 安装

go get github.com/codeudan/k

或者

从 release 下载可执行文件来直接使用


## 开发
git clone

go run k.go

go build