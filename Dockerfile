FROM yunion/alpine-cn:3.9

MAINTAINER "Zexi Li <lizexi@yunionyun.com>"

ENV TZ Asia/Shanghai

ENV PATH="/opt/yunion/bin:${PATH}"

RUN mkdir -p /opt/yunion/bin

ADD ./_output/bin/dingtalk /opt/yunion/bin/dingtalk
ADD ./_output/bin/email /opt/yunion/bin/email
ADD ./_output/bin/smsaliyun /opt/yunion/bin/smsaliyun
ADD ./_output/bin/websocket /opt/yunion/bin/websocket
