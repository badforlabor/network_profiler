/**
 * Auth :   liubo
 * Date :   2020/7/1 10:07
 * Comment:
 */

package main

import (
	"github.com/davyxu/golog"
	"runtime/debug"
	"time"
)

func TimeNowMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
func AddId(id int32) int32 {
	return (id + 1) % 10000
}

func CheckPanic(logger *golog.Logger) {
	var err = recover()
	if err != nil {
		buff := debug.Stack()
		logger.Errorln("exception:", err, string(buff))
	}
}
