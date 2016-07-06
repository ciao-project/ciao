PREFIX ?= /usr
BINDIR ?= $(DESTDIR)$(PREFIX)/bin
LIBDIR ?= $(DESTDIR)$(PREFIX)/lib

GOPATH = "$(shell pwd)"

DOMAIN = github.com
ORG = 01org
PROJECT = ciao
IMPORTLIB = $(DOMAIN)/$(ORG)/$(PROJECT)

SUBDIRS = ciao-cli ciao-controller ciao-launcher ciao-scheduler networking/ciao-cnci-agent ciao-cert

.PHONY: build
build: src src/$(IMPORTLIB)
	@export GO15VENDOREXPERIMENT=1 ; \
	export GOPATH=$(GOPATH) ; \
	for d in $(SUBDIRS); do \
	    go build -o  "$$d"/`basename "$$d"` -v -x $(IMPORTLIB)/"$$d" ; \
	done

.PHONY: install
install:
	mkdir -p $(BINDIR) $(LIBDIR)/systemd/system $(LIBDIR)/tmpfiles.d
	for d in $(SUBDIRS); do \
	    install -D "$$d"/`basename "$$d"` $(BINDIR) ; \
	done
	install -m 0644 ./data/systemd/ciao.conf $(LIBDIR)/tmpfiles.d/ciao.conf
	install -D ./networking/ciao-cnci-agent/scripts/ciao-cnci-agent.service $(LIBDIR)/systemd/system/

.PHONY: uninstall
uninstall:
	rm -f $(BINDIR)/ciao-cli
	rm -f $(BINDIR)/ciao-controller
	rm -f $(BINDIR)/ciao-launcher
	rm -f $(BINDIR)/ciao-scheduler
	rm -f $(BINDIR)/ciao-cnci-agent
	rm -f $(BINDIR)/ciao-cert
	rm -f $(LIBDIR)/tmpfiles.d/ciao.conf
	rm -f $(LIBDIR)/systemd/system/ciao-cnci-agent.service

.PHONY: clean
clean:
	@rm -f ciao-cli/ciao-cli
	@rm -f ciao-controller/ciao-controller
	@rm -f ciao-launcher/ciao-launcher
	@rm -f ciao-scheduler/ciao-scheduler
	@rm -f networking/ciao-cnci-agent/ciao-cnci-agent
	@rm -f ciao-cert/ciao-cert
	@rm -rf src

src:
	@mkdir -p src/$(DOMAIN)/$(ORG)

src/$(IMPORTLIB):
	@ln -s ../../../ src/$(IMPORTLIB) ; \
