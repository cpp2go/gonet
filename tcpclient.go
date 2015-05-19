package gonet

import (
	"net"
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

	return conn, nil
}
