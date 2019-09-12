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

package websocket

import (
	"fmt"
	"google.golang.org/grpc"
	"net"
	"notify-plugin/pkg/apis"
	"notify-plugin/utils"
	"os"
	"os/signal"
	"syscall"

	"yunion.io/x/log"
)

func StartService() {
	// config parse:
	var config SWebsocketConfig
	utils.ParseOptions(&config, os.Args, "websocket.conf")
	log.SetLogLevelByString(log.Logger(), config.LogLevel)

	// check template and socket dir
	err := utils.CheckDir(config.TemplateDir, "content", "title")
	if err != nil {
		log.Fatalf("Dir %s not exist and create failed.", config.TemplateDir)
	}
	err = utils.CheckDir(config.SockFileDir)
	if err != nil {
		log.Fatalf("Dir %s not exist and create failed.", config.SockFileDir)
	}

	// init sender manager
	senderManager = newSSenderManager(&config)
	senderManager.updateTemplateCache()

	// init rpc Server
	grpcServer := grpc.NewServer()
	apis.RegisterSendAgentServer(grpcServer, &Server{apis.UnimplementedSendAgentServer{},"webconsole"})

	la, err := net.Listen("unix", fmt.Sprintf("%s/%s.sock", config.SockFileDir, "webconsole"))
	if err != nil {
		log.Fatalln(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
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
