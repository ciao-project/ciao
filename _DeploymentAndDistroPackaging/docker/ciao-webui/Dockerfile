FROM clearlinux
MAINTAINER obed.n.munoz@intel.com

ARG WEBUI_VERSION=1.2.8.14

ARG swupd_args

RUN swupd bundle-add network-basic nodejs-basic $swupd_args

RUN mkdir -p /etc/ciao-webui
COPY ciao_config.json /etc/ciao-webui
COPY bootstrap.sh /usr/bin/bootstrap.sh

RUN curl -LOk https://github.com/01org/ciao-webui/archive/v${WEBUI_VERSION}.tar.gz
RUN tar xf v${WEBUI_VERSION}.tar.gz
RUN mv ciao-webui-${WEBUI_VERSION} ciao-webui

CMD '/usr/bin/bootstrap.sh'
