## notify-plugin

### 概述

此项目是 [onecloud](https://github.com/yunionio/onecloud/) 其中notify组件的可插拔模块，本质上是一组RPC服务端。

现共有7个不同的模块，分别是基于 [gopkg.in/gomail.v2](https://github.com/go-gomail/gomail) 的email模块；基于阿里云短信服务的smsaliyun模块；基于 [github
.com/hugozhu/godingtalk](https://github.com/hugozhu/godingtalk) 的dingtalk模块；onecloud内部的基于websocket发送消息的模块；飞书模块；钉钉webhook机器人的模块；飞书webhook机器人的模块。

### 代码结构

cmd：主函数入口

pkg：各个模块的主要代码

pkg/apis：基于 proto buffer 的grpc 接口

pkg/common：公共的代码

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

#### 注意

采用原生 gopkg.io/mail.v2 出现无法认证通过某些采用plain text认证的mail server，修改为 github.com/yunionio/mail 后，可以通过。但是又出现无法认证通过原来可以认证通过mail server的情况，因此暂时改回使用原声 gopkg.io/mail.v2
