FROM yunion/alpine-cn:3.9

MAINTAINER "Zexi Li <lizexi@yunionyun.com>"

ENV TZ Asia/Shanghai

ENV PATH="/opt/yunion/bin:${PATH}"

RUN mkdir -p /opt/yunion/bin

ADD ./_output/bin /opt/yunion/bin
