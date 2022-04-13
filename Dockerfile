# TODO: fix this websocket copy from
FROM registry.cn-beijing.aliyuncs.com/yunionio/notify-plugins:v3.8.8 as old-plugins

FROM registry.cn-beijing.aliyuncs.com/yunionio/onecloud-base:v0.2

MAINTAINER "Zexi Li <lizexi@yunionyun.com>"

ENV TZ Asia/Shanghai

ENV PATH="/opt/yunion/bin:${PATH}"

RUN mkdir -p /opt/yunion/bin

COPY --from=old-plugins /opt/yunion/bin/websocket /opt/yunion/bin
ADD ./_output/alpine-build/bin /opt/yunion/bin/
