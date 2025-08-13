# Runtime Log

最近以 docker 镜像为产物的项目中，原本使用了 logrus 作为日志工具，实际当中，用得最多的是在客户运行环境中检查日志，希望能临时开启日志，不用的时候不用输出日志，避免占用太多磁盘空间，实际上大部分客户的容器只要输出 Error 即可，标准日志级别（Debug、Info、Warn、Error）在微服务中也完全用不上。

```sh
# 执行单元测试
go test
```

Golang 运行时动态决定是否要输出 log，输出那些 log，不用重启服务。

### 原理

rlog 输出日志只有 2 个级别：

1. Error，无论什么情况都会输出
2. Info，运行时根据条件输出

每隔 1s 检查一次运行程序同目录下的 `rlog` 文件，如果配置了 `*`，那么 Info 级别信息就会输出，否则就只有 Error 输出。

### 安装

在自己的项目中创建 `pkg/rlog` 目录，然后把这个项目中的内容都复制到新建目录里，然后随意使用。

或者

```sh
go get github.com/delong8/rlog
```

### 基础用法

直接使用，Error 都输出，Info 看情况

```go
package main

import "github.com/delong8/rlog"

func main(){
    i := 0
    for {
        time.Sleep(time.Second)
        i += 1
        rlog.Info("info", i)
        rlog.Error("error", i)
    }
}
```

运行结果

error 1
error 2
error 3
...

这时在运行同级目录（注意是启动命令的统计目录），创建 rlog 文件，写入 “\*” ，然后保存。上面正在运行中的程序，**不需要**重启，日志输出会变成

info 10
error 10
info 11
error 11
info 12
error 12
...

```go
package main

import "github.com/delong8/rlog"

func main(){

}
```

### 高级用法

设置日志范围，先创建带名字的 logger，然后想输出哪些日志，就在 rlog 中添加 logger 的名字。

```go
package main

import "github.com/delong8/rlog"

func main(){
    la := rlog.New("a")
    lb := rlog.New("b")
    lab := rlog.New("a.b")

    i := 0
    for {
        time.Sleep(time.Second)
        i += 1
        la.Info("a", i)
        lb.Info("b", i)
        lab.Info("a.b", i)
    }
}
```

| rlog 内容 | 输入内容                |
| --------- | ----------------------- |
| \*        | a 1<br />b 1<br />a.b 1 |
| a         | a 1<br />a.b 1          |
| b         | b 1                     |
| a.b       | a.b 1                   |
| a<br />b  | a 1<br />b 1<br />a.b 1 |

### 参数说明

可以手动执行 `rlog.Init` 来进行全局设置，在调用其它方法前设置且仅设置一次。

```go
cfg := &rlog.Config{
    File: "/app/log_config",
    Read: func(){
        return "*"
    },
    Interval: time.Duration(time.Minute),
    Print: func(v ...any){
        fmt.Println(v)
    },
}
rlog.Init(cfg)
```

| 参数     | 默认值                                        | 说明                         |
| -------- | --------------------------------------------- | ---------------------------- |
| File     | "./rlog"                                      | 用于配置日志输出范围的文件   |
| Read     | os.Read(File)                                 | 查询日志输出范围时调用的方法 |
| Interval | 1s                                            | 获取日志范围变化的时间间隔   |
| Print    | [log.Println](https://pkg.go.dev/log#Println) | 日志输出位置                 |

参数被调用顺序说明

1. 首先根据 Interval 参数，每隔固定时间调用一次 Read 方法
   Read 说明
2. 如果 Read 参数未配置，则默认会去读 File 文件内容
3. 如果 File 参数未配置，默认在运行项目的目录寻找 rlog 文件
4. 将 rlog 文件中的内容按行分隔，每一行表示允许输出的日志的 logger 的名字。包中不需要初始化的 `rlog.Info` 方法对应的名字是 "default"
5. 当通过 `rlog.New` 方法创建出来的 logger 调用了 Info 方法时，先去查看 logger 的 name 是否前缀包含 Read 方法返回的列表中的任意一项，如果满足条件，则输出内容，否则不输出。
6. 所有输出内容会传给 Print 参数配置的方法。
