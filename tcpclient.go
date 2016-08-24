package gonet

import (
	"net"
	"time"
)

type TcpClient struct {
}

func (this *TcpClient) Connect(address string) (*net.TCPConn, error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(1 * time.Minute)
	conn.SetNoDelay(true)
	conn.SetWriteBuffer(128 * 1024)
	conn.SetReadBuffer(128 * 1024)

	return conn, nil
}
