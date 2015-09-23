package gonet

import (
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type ITcpTask interface {
	ParseMsg(data []byte, flag byte) bool
	OnClose()
}

const (
	cmd_max_size    = 128 * 1024
	cmd_header_size = 4 // 3字节指令长度 1字节是否压缩
	cmd_verify_time = 10
)

type TcpTask struct {
	closed     int32
	verified   bool
	stopedChan chan struct{}
	recvBuff   *ByteBuffer
	sendBuff   *ByteBuffer
	sendMutex  sync.Mutex
	sendChan   chan bool
	Conn       net.Conn
	Derived    ITcpTask
}

func NewTcpTask(conn net.Conn) *TcpTask {
	return &TcpTask{
		closed:     -1,
		verified:   false,
		Conn:       conn,
		stopedChan: make(chan struct{}),
		recvBuff:   NewByteBuffer(),
		sendBuff:   NewByteBuffer(),
		sendChan:   make(chan bool),
	}
}

func (this *TcpTask) Start() {
	if atomic.CompareAndSwapInt32(&this.closed, -1, 0) {
		fmt.Println("[连接] 收到连接 ", this.Conn.RemoteAddr())
		go this.sendloop()
		go this.recvloop()
	}
}

func (this *TcpTask) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		fmt.Println("[连接] 断开连接 ", this.Conn.RemoteAddr())
		this.Conn.Close()
		close(this.stopedChan)
		close(this.sendChan)
		this.Derived.OnClose()
	}
}

func (this *TcpTask) IsClosed() bool {
	return atomic.LoadInt32(&this.closed) != 0
}

func (this *TcpTask) Verify() {
	this.verified = true
}

func (this *TcpTask) IsVerified() bool {
	return this.verified
}

func (this *TcpTask) AsyncSend(buffer []byte, flag byte) bool {
	if this.IsClosed() {
		return false
	}
	bsize := len(buffer)
	this.sendMutex.Lock()
	this.sendBuff.Append(byte(bsize), byte(bsize>>8), byte(bsize>>16), flag)
	this.sendBuff.Append(buffer...)
	this.sendMutex.Unlock()
	select {
	case this.sendChan <- true:
	default:
	}
	return true
}

func (this *TcpTask) recvloop() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("[异常] ", err, "\n", string(debug.Stack()))
		}
	}()
	defer this.Close()

	var (
		neednum   int
		readnum   int
		err       error
		totalsize int
		datasize  int
		msgbuff   []byte
	)

	for {
		totalsize = this.recvBuff.RdSize()

		if totalsize < cmd_header_size {

			neednum = cmd_header_size - totalsize
			if this.recvBuff.WrSize() < neednum {
				this.recvBuff.WrGrow(neednum)
			}

			readnum, err = io.ReadAtLeast(this.Conn, this.recvBuff.WrBuf(), neednum)
			if err != nil {
				fmt.Println("[连接] 接收失败 ", this.Conn.RemoteAddr(), ",", err)
				return
			}

			this.recvBuff.WrFlip(readnum)
			totalsize = this.recvBuff.RdSize()
		}

		msgbuff = this.recvBuff.RdBuf()

		datasize = int(msgbuff[0]) | int(msgbuff[1])<<8 | int(msgbuff[2])<<16
		if datasize > cmd_max_size {
			fmt.Println("[连接] 数据超过最大值 ", this.Conn.RemoteAddr(), ",", datasize)
			return
		}

		if totalsize < cmd_header_size+datasize {

			neednum = cmd_header_size + datasize - totalsize
			if this.recvBuff.WrSize() < neednum {
				this.recvBuff.WrGrow(neednum)
			}

			readnum, err = io.ReadAtLeast(this.Conn, this.recvBuff.WrBuf(), neednum)
			if err != nil {
				fmt.Println("[连接] 接收失败 ", this.Conn.RemoteAddr(), ",", err)
				return
			}

			this.recvBuff.WrFlip(readnum)
			msgbuff = this.recvBuff.RdBuf()
		}

		this.Derived.ParseMsg(msgbuff[cmd_header_size:cmd_header_size+datasize], msgbuff[3])
		this.recvBuff.RdFlip(cmd_header_size + datasize)
	}
}

func (this *TcpTask) sendloop() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("[异常] ", err, "\n", string(debug.Stack()))
		}
	}()
	defer this.Close()

	var (
		tmpByte  = NewByteBuffer()
		timeout  = time.NewTimer(time.Second * cmd_verify_time)
		writenum int
		err      error
	)

	defer timeout.Stop()

	for {
		select {
		case <-this.sendChan:
			for {
				this.sendMutex.Lock()
				if this.sendBuff.RdReady() {
					tmpByte.Append(this.sendBuff.RdBuf()[:this.sendBuff.RdSize()]...)
					this.sendBuff.Reset()
				}
				this.sendMutex.Unlock()
				if !tmpByte.RdReady() {
					break
				}
				writenum, err = this.Conn.Write(tmpByte.RdBuf()[:tmpByte.RdSize()])
				if err != nil {
					fmt.Println("[连接] 发送失败 ", this.Conn.RemoteAddr(), ",", err)
					return
				}
				tmpByte.RdFlip(writenum)
			}
		case <-this.stopedChan:
			return
		case <-timeout.C:
			if !this.IsVerified() {
				fmt.Println("[连接] 验证超时 ", this.Conn.RemoteAddr())
				return
			}
		}
	}
}
