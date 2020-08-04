// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func StartService(opt IServiceOptions, generator func(IServiceOptions) ISender, service string, configFile string) {
	// config parse:
	ParseOptions(opt, os.Args, configFile)
	log.SetLogLevelByString(log.Logger(), opt.GetLogLevel())
	log.Infof("opt: %#v", opt.(*SBaseOptions))

	// check socket dir
	err := CheckDir(opt.GetSockFileDir())
	if err != nil {
		log.Fatalf("Dir %s not exist and create failed.", opt.GetSockFileDir())
	}

	// init rpc Server
	grpcServer := grpc.NewServer()
	apis.RegisterSendAgentServer(grpcServer, NewServer(generator(opt)))

	socketFile := fmt.Sprintf("%s/%s.sock", opt.GetSockFileDir(), service)
	log.Infof("Socket file path: %s", socketFile)
	if IsExist(socketFile) {
		log.Infof("socket file already exists, try deleting...")
		err := os.Remove(socketFile)
		if err != nil {
			log.Fatalf("delete failed")
		}
		log.Infof("delete successfully")
	}
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
