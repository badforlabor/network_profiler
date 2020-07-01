/**
 * Auth :   liubo
 * Date :   2020/7/1 13:29
 * Comment:
 */

package main

import (
	"github.com/davyxu/golog"
	"strings"
)

type IDevice interface {
	Close()
}

type IServer interface {
	IDevice
	OpenServer(addr string)

}

type IClient interface {
	IDevice
	OpenClient(serverAddr string)
}

func NewClient(protocol string) IClient {
	protocol = strings.ToLower(protocol)

	return &NetClient{Protocol:protocol, Processor:protocol + ".ltv"}
}

func NewServer(protocol string) IServer {
	protocol = strings.ToLower(protocol)

	return &NetServer{Protocol:protocol, Processor:protocol + ".ltv"}
}


var netLog = golog.New("net")

func recordAck(host string, old, msg *PtAck) {
	if msg.Id == old.Id {
		if old.Time != msg.Time {
			netLog.Warnln("收到的协议是错误的！", old.Id, host)
		} else {
			var delta = TimeNowMs() - msg.Time
			if delta > globalConfig.MaxWaitTime {
				netLog.Warnf("收到协议返回，超时了, id=%d, cost(ms)=%d, host=%s\n", old.Id, delta, host)
			} else {
				netLog.Infof("收到协议返回, id=%d, cost(ms)=%d, host=%s\n", old.Id, delta, host)
			}
		}
	} else {
		var delta = TimeNowMs() - msg.Time
		if delta > globalConfig.MaxWaitTime {
			netLog.Warnf("协议错乱，超时了, id=%d, cost(ms)=%d, host=%s\n", msg.Id, delta, host)
		} else {
			netLog.Infof("协议错乱，id=%d, cost(ms)=%d, host=%s\n", msg.Id, delta, host)
		}
	}
}