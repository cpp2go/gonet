package gonet

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
)

type IService interface {
	Init() bool
	MainLoop()
	Final() bool
}

type Service struct {
	terminate bool
	Derived   IService
}

func (this *Service) Terminate() {
	this.terminate = true
}

func (this *Service) isTerminate() bool {
	return this.terminate
}

func (this *Service) Main() bool {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("[异常] ", err, "\n", string(debug.Stack()))
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGPIPE)
	go func() {
		for sig := range ch {
			switch sig {
			case syscall.SIGPIPE:
			default:
				this.Terminate()
			}
			fmt.Println("[服务] 收到信号 ", sig)
		}
	}()

	runtime.GOMAXPROCS(runtime.NumCPU())

	if !this.Derived.Init() {
		return false
	}

	for !this.isTerminate() {
		this.Derived.MainLoop()
	}

	this.Derived.Final()
	return true
}
