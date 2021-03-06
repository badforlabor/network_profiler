/**
 * Auth :   liubo
 * Date :   2020/7/1 13:30
 * Comment: TCP测试
 */

package main

import (
	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/peer"
	"github.com/davyxu/cellnet/proc"
	"github.com/davyxu/cellnet/timer"
	"github.com/davyxu/cellnet/util"
	"github.com/davyxu/golog"
	"time"

	_ "github.com/davyxu/cellnet/peer/tcp"
	_ "github.com/davyxu/cellnet/proc/tcp"

	_ "github.com/davyxu/cellnet/peer/udp"
	_ "github.com/davyxu/cellnet/proc/udp"
)

var netLog = golog.New("net")

type NetServer struct {
	Protocol string
	Processor string

	queue cellnet.EventQueue
	peer cellnet.GenericPeer
}

func (self *NetServer) OpenServer(addr string) {

	netLog.Infoln("open server:", addr)

	// 创建一个事件处理队列，整个服务器只有这一个队列处理事件，服务器属于单线程服务器
	queue := cellnet.NewEventQueue()
	self.queue = queue

	// 创建一个tcp的侦听器，名称为server，连接地址为127.0.0.1:8801，所有连接将事件投递到queue队列,单线程的处理（收发封包过程是多线程）
	// addr = "127.0.0.1:8801"
	p := peer.NewGenericPeer( self.Protocol + ".Acceptor", self.Protocol + ".server", addr, queue)
	self.peer = p

	// tcp设置读写5秒超时
	var tcp, ok = p.(cellnet.TCPAcceptor)
	if ok {
		tcp.SetSocketDeadline(time.Second*5, time.Second*5)
	}

	// 设定封包收发处理的模式为tcp的ltv(Length-Type-Value), Length为封包大小，Type为消息ID，Value为消息内容
	// 每一个连接收到的所有消息事件(cellnet.Event)都被派发到用户回调, 用户使用switch判断消息类型，并做出不同的处理
	proc.BindProcessorHandler(p, self.Processor, self.onMsg)

	// 开始侦听
	p.Start()

	// 事件队列开始循环
	queue.StartLoop()

	// 阻塞等待事件队列结束退出( 在另外的goroutine调用queue.StopLoop() )
	// queue.Wait()

}
func (self *NetServer) Close() {
	self.queue.StopLoop()
}

func (self *NetServer) onMsg(ev cellnet.Event) {

	defer CheckPanic(netLog)

	switch msg := ev.Message().(type) {
	// 有新的连接
	case *cellnet.SessionAccepted:
		netLog.Debugln("server accepted")
	// 有连接断开
	case *cellnet.SessionClosed:
		netLog.Debugln("session closed: ", ev.Session().ID())

	case *PtAck:

		var ret = *msg
		//var s = ev.Session().Raw()
		//var remoteAddr = s.(net.Conn).RemoteAddr().String()
		var remoteAddr, _ = util.GetRemoteAddrss(ev.Session())
		netLog.Infof("收到信息, from=[%s], msg=[%d]", remoteAddr, ret.Id)
		ev.Session().Send(ret)
	}
}

type NetClient struct {
	Protocol string
	Processor string

	host string

	queue cellnet.EventQueue
	peer  cellnet.GenericPeer
	session cellnet.Session

	loopCheck1Second *timer.Loop

	lastAck PtAck
	lastRcvTime int64

	// udp没有重连机制，改成手动的
	udpDisconnectCount  int
}
func (self *NetClient) OpenClient(addr string) {
	netLog.Infoln("open client. host:", addr, self.Protocol, self.Processor)

	self.host = addr

	// 创建一个事件处理队列，整个客户端只有这一个队列处理事件，客户端属于单线程模型
	queue := cellnet.NewEventQueue()
	self.queue = queue

	// 创建一个tcp的连接器，名称为client，连接地址为127.0.0.1:8801，将事件投递到queue队列,单线程的处理（收发封包过程是多线程）
	p := peer.NewGenericPeer(self.Protocol + ".Connector", self.Protocol + ".client", addr, queue)
	self.peer = p

	// 设置重连
	var tcp, ok = p.(cellnet.TCPConnector)
	if ok {
		tcp.SetReconnectDuration(time.Second * 5)
	}

	// 设定封包收发处理的模式为tcp的ltv(Length-Type-Value), Length为封包大小，Type为消息ID，Value为消息内容
	// 并使用switch处理收到的消息
	proc.BindProcessorHandler(p, self.Processor, self.onMsg)


	// 开始发起到服务器的连接
	p.Start()
	netLog.Infoln(self.Protocol + " client start...")

	self.loopCheck1Second = timer.NewLoop(queue, time.Second, func(loop *timer.Loop) {
		self.timeEvery1Second()
	}, nil)
	self.loopCheck1Second.Start()

	// 事件队列开始循环
	queue.StartLoop()
}
func (self *NetClient) timeEvery1Second() {
	self.lastAck.Id = AddId(self.lastAck.Id)
	self.lastAck.Time = TimeNowMs()
	if len(self.lastAck.Stuffing) != globalConfig.StuffingCount && globalConfig.StuffingCount > 0{
		self.lastAck.Stuffing = make([]int32, globalConfig.StuffingCount)
	}
	if len(self.lastAck.Stuffing) > 0 {
		self.lastAck.Stuffing[len(self.lastAck.Stuffing)-1] = self.lastAck.Id
	}

	var msg = self.lastAck
	if self.session != nil {
		self.session.Send(&msg)
	} else {
		netLog.Warnln("网络断开了，无法发包:", self.lastAck.Id)
		disconnectCount++
	}

	// 超过1.5秒没有收到数据包
	if self.lastRcvTime > 0 && TimeNowMs() - self.lastRcvTime > 1500 {
		netLog.Warnln("网络断开了，无法收到包")
		disconnectCount++

		self.udpDisconnectCount++
		if self.udpDisconnectCount > 10 {
			self.udpDisconnectCount = 0
			if self.Protocol == "udp" {
				self.peer.Stop()
				go func() {
					time.Sleep(time.Second)
					self.peer.Start()
				}()
			}
		}
	} else {
		self.udpDisconnectCount = 0
	}
}
func (self *NetClient) onMsg(ev cellnet.Event) {

	defer CheckPanic(netLog)

	switch msg := ev.Message().(type) {
	case *cellnet.SessionConnected:
		self.session = ev.Session()
		netLog.Infoln("client connected")
	case *cellnet.SessionClosed:
		self.session = nil
		netLog.Infoln("client error")
	case *PtAck:
		recordAck(self.host, &self.lastAck, msg)
		self.lastRcvTime = TimeNowMs()
	}
}
func (self *NetClient) Close() {
	self.peer.Stop()
}

func recordAck(host string, old, msg *PtAck) {

	if len(msg.Stuffing) > 0 {
		if msg.Stuffing[len(msg.Stuffing) - 1] != msg.Id {
			netLog.Warnln("收到的协议是错误的！", old.Id, host)
			errCount++
		}
	}

	if msg.Id == old.Id {
		if old.Time != msg.Time {
			netLog.Warnln("收到的协议是错误的！", old.Id, host)
			errCount++
		} else {
			var delta = TimeNowMs() - msg.Time
			if delta > globalConfig.MaxWaitTime {
				netLog.Warnf("收到协议返回，超时了, id=%d, cost(ms)=%d, host=%s\n", old.Id, delta, host)
				overtimeCount++
			} else {
				netLog.Infof("收到协议返回, id=%d, cost(ms)=%d, host=%s\n", old.Id, delta, host)
			}
		}
	} else {
		var delta = TimeNowMs() - msg.Time
		if delta > globalConfig.MaxWaitTime {
			netLog.Warnf("协议错乱，超时了, id=%d, cost(ms)=%d, host=%s\n", msg.Id, delta, host)
			errCount++
			overtimeCount++
		} else {
			netLog.Warnf("协议错乱，id=%d, cost(ms)=%d, host=%s\n", msg.Id, delta, host)
			errCount++
		}
	}
}