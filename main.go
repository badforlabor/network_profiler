/**
 * Auth :   liubo
 * Date :   2020/7/1 9:12
 * Comment:
 */

package main

import (
	"flag"
	"github.com/davyxu/golog"
	"gopkg.in/ini.v1"
	"os"
	"os/signal"
	"strconv"
)

type GlobalConfig struct {

	// 最大等待时间（毫秒），超过这个时间点额，将记录在日志中
	MaxWaitTime int64

	// 协议类型(TCP, UDP)
	Proto string

	// 客户端，服务器
	ServerAddr string

	// 是否是客户端（1是服务器，2是客户端）
	Role ERole

	// 最多多少个日志
	LogMaxCount int

	// 每个数据包额外带多少数据
	StuffingCount int
}

type ERole int32
const (
	ERoleNone ERole = iota
	ERoleClient
	ERoleServer
)

var globalConfig GlobalConfig

func main() {
	flag.Parse()

	defer CheckPanic(netLog)

	// 配置文件
	var iniCfg, err = ini.Load("config.ini")
	if err == nil {
		err = iniCfg.Section("main").MapTo(&globalConfig)
	}
	if err != nil {
		var cfg = ini.Empty()
		cfg.Section("main").ReflectFrom(&globalConfig)
		cfg.SaveTo("config.ini")
		panic("读取配置文件错误!")
		return
	}

	if globalConfig.LogMaxCount > 0 {
		MaxKeepLogCount = globalConfig.LogMaxCount
	}

	// 日志
	var w = CrazyLogWriter("logs", "net-" + globalConfig.Proto + "-" + strconv.Itoa(int(globalConfig.Role)), true)

	golog.VisitLogger(".*", func(logger *golog.Logger) bool {
		logger.SetLevel(golog.Level_Info)
		logger.SetOutptut(w)
		return true
	})

	netLog.Infoln("配置文件:", globalConfig)

	var worker IDevice

	if globalConfig.Role == ERoleClient {
		var client = NewClient(globalConfig.Proto)
		client.OpenClient(globalConfig.ServerAddr)
		worker = client
	} else if globalConfig.Role == ERoleServer {
		var server = NewServer(globalConfig.Proto)
		server.OpenServer(globalConfig.ServerAddr)
		worker = server
	} else {
		panic("无效的配置文件")
	}

	// 监听终止
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	netLog.Infoln("等待中止信号")
	<-c
	netLog.Infoln("收到中止信号")

	worker.Close()

}


