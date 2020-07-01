/**
 * Auth :   liubo
 * Date :   2020/7/1 15:11
 * Comment: 在日志文件夹，最多保存N个日志，并且每个日志大小超过M后，会切割文件
 */

package main

import (
	"fmt"
	"github.com/badforlabor/gocrazy/alg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var MaxLogSize int64 = 64 * 1000 * 1000	//64M
var MaxKeepLogCount = 30
var LogExt = ".log"

type logData struct {
	data []byte
}

type rollingLogWriter struct {
	folder     string
	fileName   string

	channel chan []byte
	quit chan bool

	fileHandle *os.File // 当前写入的日志文件句柄
	fileSize   int64
	mutex      sync.Mutex
}
func (self *rollingLogWriter) Quit() {
	self.quit <- true
}
func (self *rollingLogWriter) init(folderName, fileName string) {

	self.folder = folderName
	self.fileName = fileName

	self.checkFile()

	self.channel = make(chan []byte, 100)
	self.quit = make(chan bool)

	go func() {
		var quit = false
		for !quit {
			select {
				case d := <-self.channel:

					if self.fileHandle != nil {
						self.fileHandle.Write(d)
					}
					self.checkRolling(len(d))

				case <-self.quit:
					fmt.Println("收到消息，准备退出写文件日志")
					quit = true
			}
		}
		fmt.Println("退出写文件日志")
	}()
}
func (self *rollingLogWriter) checkFile() {

	self.mutex.Lock()
	defer self.mutex.Unlock()

	if self.fileHandle != nil {
		return
	}

	var folder, err = os.Stat(self.folder)
	if err != nil || folder == nil {
		os.MkdirAll(self.folder, os.ModePerm)
	}

	folder, err = os.Stat(self.folder)
	if folder == nil {
		fmt.Errorf("\n无法创建文件夹%s\n", folder)
		return
	}

	{

		// 检查本文件夹，如果超过10个文件，那么删除掉
		var allfile []os.FileInfo
		filepath.Walk(self.folder, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				if info.Name() == folder.Name() {
					return nil
				}
				return filepath.SkipDir
			}

			if strings.HasSuffix( info.Name(), LogExt) {
				allfile = append(allfile, info)
			}

			return nil
		})

		if len(allfile) >= MaxKeepLogCount {
			alg.Sort(&allfile, func(left interface{}, right interface{}) bool {
				var a = left.(os.FileInfo)
				var b = right.(os.FileInfo)
				return a.ModTime().Unix() < b.ModTime().Unix()
			})
		}

		for len(allfile) >= MaxKeepLogCount {
			var f = allfile[0]
			os.Remove(filepath.Join(self.folder, f.Name()))
			allfile = allfile[1:]
		}

		var n int64 = 0
		for _, v := range allfile {
			var trim = v.Name()[0:len(v.Name()) - len(LogExt)]
			var idx = strings.LastIndexByte(trim, '.')
			if idx >= 0 {
				var nn, _ = strconv.ParseInt(trim[idx+1:], 10, 0)
				if nn > n {
					n = nn
				}
			}
		}

		n = n + 1

		var filename = filepath.Join(self.folder, self.fileName + "." + strconv.Itoa(int(n)) + ".log")
		self.fileHandle, _ = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if self.fileHandle != nil {

			fmt.Println("创建文件成功:", filename)

			var s, _ = self.fileHandle.Stat()
			self.fileSize = s.Size()
		} else {
			self.fileSize = 0
		}
	}


	if self.fileHandle == nil {
		fmt.Errorf("\n无法创建文件%s\n", filepath.Join(self.fileName))
	}
}
func (self *rollingLogWriter) checkRolling(n int) {

	atomic.AddInt64(&self.fileSize, int64(n))

	if self.fileSize > MaxLogSize {

		self.mutex.Lock()
		if self.fileHandle != nil {
			self.fileHandle.Sync()
			self.fileHandle.Close()
			self.fileHandle = nil
		}
		self.mutex.Unlock()

		self.checkFile()
	}
}

func (self *rollingLogWriter)Write(p []byte) (n int, err error) {

	self.channel <- p

	return len(p), nil
}

func CrazyLogWriter(folderName, fileName string, toConsole bool) io.Writer {

	if len(LogExt) == 0 {
		panic("err log extension.")
	}

	var r rollingLogWriter

	r.init(folderName, fileName)

	if !toConsole {
		return &r
	}

	return io.MultiWriter(&r, os.Stdout)
}
