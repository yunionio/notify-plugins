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
	"net/rpc"
	"notify-plugin/utils"
	"os"
	"os/signal"
	"syscall"

	"yunion.io/x/log"
)

var senderManager *sSenderManager

func StartService() {
	// config parse:
	var config SRegularConfig
	ParseOptions(&config, os.Args, "email.conf")

	// check template and socket dir
	err := utils.CheckDir(config.TemplateDir)
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
	rpcServer := rpc.NewServer()
	server := &Server{
		name: "email",
	}
	rpcServer.Register(server)
	la, e := net.Listen("unix", fmt.Sprintf("%s/%s.sock", config.SockFileDir, "email"))
	if e != nil {
		log.Errorf("rpc server start failed because that %s.", e.Error())
		return
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go rpcServer.Accept(la)
	log.Infoln("Service start successfully")

	//tmp := make(chan struct{})
	//go func(){
	//	wg.Wait()
	//	close(tmp)
	//}()

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
