package gonet

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type ITcpTask interface {
	ParseMsg(data []byte) bool
	OnClose()
}

const (
	cmd_max_size    = 128 * 1024
	cmd_header_size = 4 // 3字节指令长度 1字节是否压缩
	cmd_verify_time = 10
	cmd_zip_size    = 1024
)

type TcpTask struct {
	closed     int32
	verified   bool
	stopedChan chan struct{}
	recvBuff   *ByteBuffer
	sendBuff   *ByteBuffer
	sendMutex  sync.Mutex
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
	}
}

func zlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	defer w.Close()
	_, err := w.Write(src)
	if err != nil {
		return nil
	}
	return in.Bytes()
}

func zlibUnCompress(src []byte) []byte {
	b := bytes.NewReader(src)
	var out bytes.Buffer
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil
	}
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil
	}
	return out.Bytes()
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

func (this *TcpTask) AsyncSend(msg []byte) bool {
	if this.IsClosed() {
		return false
	}
	var (
		mbuffer    []byte
		msize      int
		iscompress byte
	)
	if len(msg) > cmd_zip_size {
		mbuffer = zlibCompress(msg)
		if mbuffer == nil {
			return false
		}
		iscompress = 1
	} else {
		mbuffer = msg
		iscompress = 0
	}
	msize = len(mbuffer)
	this.sendMutex.Lock()
	this.sendBuff.Append(byte(msize), byte(msize>>8), byte(msize>>16), iscompress)
	this.sendBuff.Append(mbuffer...)
	this.sendMutex.Unlock()
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
		writable  int
		readnum   int
		err       error
		totalsize int
		datasize  int
		msgbuff   []byte
		mbuffer   []byte
	)

	for {
		writable = this.recvBuff.WrSize()
		if writable > 0 {
			readnum, err = this.Conn.Read(this.recvBuff.WrBuf())
			if err != nil {
				fmt.Println("[连接] 接收失败 ", this.Conn.RemoteAddr(), ",", err)
				return
			}
			this.recvBuff.WrFlip(readnum)
		} else {
			tmpbuff := make([]byte, 64*1024)
			readnum, err = this.Conn.Read(tmpbuff)
			if err != nil {
				fmt.Println("[连接] 接收失败 ", this.Conn.RemoteAddr(), ",", err)
				return
			}
			this.recvBuff.Append(tmpbuff[:readnum]...)
		}

		for {
			totalsize = this.recvBuff.RdSize()
			if totalsize < cmd_header_size {
				break
			}
			msgbuff = this.recvBuff.RdBuf()
			datasize = int(msgbuff[0]) | int(msgbuff[1]<<8) | int(msgbuff[2]<<16)
			if datasize > cmd_max_size {
				fmt.Println("[连接] 数据超过最大值 ", this.Conn.RemoteAddr(), ",", datasize)
				return
			}
			if totalsize < datasize+cmd_header_size {
				break
			}
			if msgbuff[3] != 0 {
				mbuffer = zlibUnCompress(msgbuff[cmd_header_size : datasize+cmd_header_size])
			} else {
				mbuffer = msgbuff[cmd_header_size : datasize+cmd_header_size]
			}
			this.Derived.ParseMsg(mbuffer)
			this.recvBuff.RdFlip(datasize + cmd_header_size)
		}
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
		tick     = time.Tick(time.Millisecond * 10)
		timeout  = time.After(time.Second * cmd_verify_time)
		writenum int
		err      error
	)

	for {
		select {
		case <-tick:
			{
				this.sendMutex.Lock()
				if this.sendBuff.RdReady() {
					tmpByte.Append(this.sendBuff.RdBuf()[:this.sendBuff.RdSize()]...)
					this.sendBuff.Reset()
				}
				this.sendMutex.Unlock()

				if tmpByte.RdReady() {
					writenum, err = this.Conn.Write(tmpByte.RdBuf()[:tmpByte.RdSize()])
					if err != nil {
						fmt.Println("[连接] 发送失败 ", this.Conn.RemoteAddr(), ",", err)
						return
					}
					tmpByte.RdFlip(writenum)
				}
			}
		case <-this.stopedChan:
			return
		case <-timeout:
			if !this.IsVerified() {
				fmt.Println("[连接] 验证超时 ", this.Conn.RemoteAddr())
				return
			}
		}
	}
}
