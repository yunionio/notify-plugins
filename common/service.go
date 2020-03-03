package common

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"yunion.io/x/log"

	"yunion.io/x/notify-plugin/pkg/apis"
)

func StartService(opt IServiceOptions, srv apis.SendAgentServer, service string, configFile string, init func()) {
	// config parse:
	ParseOptions(opt, os.Args, configFile)
	log.Debugf("sendnum: %d", opt.(*SBaseOptions).SenderNum)
	log.SetLogLevelByString(log.Logger(), opt.GetLogLevel())

	// check socket dir
	err := CheckDir(opt.GetSockFileDir())
	if err != nil {
		log.Fatalf("Dir %s not exist and create failed.", opt.GetSockFileDir())
	}

	// init
	init()

	// init rpc Server
	grpcServer := grpc.NewServer()
	apis.RegisterSendAgentServer(grpcServer, srv)

	socketFile := fmt.Sprintf("%s/%s.sock", opt.GetSockFileDir(), service)
	log.Infof("Socket file path: %s", socketFile)
	la, err := net.Listen("unix", socketFile)
	if err != nil {
		log.Fatalln(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go grpcServer.Serve(la)
	log.Infoln("Service start successfully")

	select {
	//case <-tmp:
	//	log.Errorln("All sender quit.")
	//	la.Close()
	case <-sigs:
		log.Println("Receive stop signal, stopping....")
		la.Close()
		log.Println("Stopped")
	}
}
