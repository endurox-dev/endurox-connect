export GOPATH=$(shell pwd)

all:
	go get golang.org/x/net/netutil
	$(MAKE) -C src

clean:
	rm -rf pkg bin
	$(MAKE) -C src clean


.PHONY: clean all
