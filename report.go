/**
 * Auth :   liubo
 * Date :   2020/7/2 13:00
 * Comment: 汇报
 */

package main

import (
	"fmt"
	"github.com/davyxu/cellnet/util"
	"time"
)


var disconnectCount int // 网络断开了
var errCount int        // 协议错乱了
var overtimeCount int   // 协议超时了

func timerReportData() {

	var localIp = util.GetLocalIP()

	for true {
		time.Sleep(10 * time.Second)
		if disconnectCount > 0 || errCount > 0 || overtimeCount > 0 {
			func() {
				CheckPanic(netLog)
				var subject = "network-profiler:" + localIp
				var body = fmt.Sprintf(globalConfig.Proto + " 断网:%d, 协议错乱:%d, 超时:%d", disconnectCount, errCount, overtimeCount)
				body += ". 服务器是：" + globalConfig.ServerAddr
				var succ = true

				if globalConfig.NotEmail == 0 {
					succ = sendEmail(subject, body)
				}

				if succ {
					disconnectCount = 0
					errCount = 0
					overtimeCount = 0
				}
			}()
		}
	}

}