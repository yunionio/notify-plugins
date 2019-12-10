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

package email

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"yunion.io/x/log"

	"notify-plugin/pkg/apis"
	"notify-plugin/utils"
)

var senderManager *sSenderManager

func StartService() {
	// config parse:
	var config SEmailConfig
	utils.ParseOptions(&config, os.Args, "email.conf")
	log.SetLogLevelByString(log.Logger(), config.LogLevel)

	// check socket dir
	err := utils.CheckDir(config.SockFileDir)
	if err != nil {
		log.Fatalf("Dir %s not exist and create failed.", config.SockFileDir)
	}
	// init sender manager
	senderManager = newSSenderManager(&config)

	// init rpc Server
	grpcServer := grpc.NewServer()
	apis.RegisterSendAgentServer(grpcServer, &Server{apis.UnimplementedSendAgentServer{}, "email"})

	la, err := net.Listen("unix", fmt.Sprintf("%s/%s.sock", config.SockFileDir, "email"))
	if err != nil {
		log.Fatalln(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go grpcServer.Serve(la)
	log.Infoln("Service start successfully")

	select {
	case <-sigs:
		log.Println("Receive stop signal, stopping....")
		la.Close()
		log.Println("Stopped")
	}
}
