package main

import (
	"fmt"
	"gonet"
	"time"
)

type Client struct {
	gonet.TcpTask
	mclient *gonet.TcpClient
}

func NewClient() *Client {
	s := &Client{
		TcpTask: *gonet.NewTcpTask(nil),
	}
	s.Derived = s
	return s
}

func (this *Client) Connect(addr string) bool {

	conn, err := this.mclient.Connect(addr)
	if err != nil {
		fmt.Println("连接失败 ", addr)
		return false
	}

	this.Conn = conn

	this.Start()

	fmt.Println("连接成功 ", addr)
	return true
}

func (this *Client) ParseMsg(data []byte) bool {

	this.Verify()

	fmt.Println(string(data))

	this.AsyncSend(data)

	return true
}

func (this *Client) OnClose() {

}

func main() {

	for {

		client := NewClient()

		if !client.Connect("127.0.0.1:80") {
			return
		}

		client.AsyncSend([]byte("hello...."))

		time.Sleep(time.Second * 1)
	}
}
