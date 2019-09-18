## notify-plugin

### 概述

此项目是 [onecloud](https://github.com/yunionio/onecloud/) 其中notify组件的可插拔模块，本质上是一组RPC服务端。

现共有4个不同的模块，分别是基于 [gopkg.in/gomail.v2](https://github.com/go-gomail/gomail) 的email模块；基于阿里云短信服务的smsaliyun模块；基于 [github
.com/hugozhu/godingtalk](https://github.com/hugozhu/godingtalk) 的dingtalk模块；onecloud内部的基于websocket发送消息的模块。

### 代码结构

cmd：主函数入口

pkg：各个模块的主要代码

pkg/apis：基于 proto buffer 的grpc 接口

utils：工具包

example:  rpc 客户端的例子，暂不可用

### 编译

```shell
# All make
make
# Separate make
make cmd/email
make cmd/smsaliyun
make cmd/dingtalk
make cmd/websocket
```

