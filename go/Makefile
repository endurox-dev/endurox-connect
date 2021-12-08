export GOPATH=$(shell pwd)
export GO111MODULE=auto

all:
	$(MAKE) -C src

clean:
	rm -rf pkg bin
	$(MAKE) -C src clean


.PHONY: clean all
