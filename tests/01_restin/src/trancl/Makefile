SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY=trancl
LDFLAGS=

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build ${LDFLAGS} -o ${BINARY} *.go

.PHONY: install
install:
	go install ${LDFLAGS} ./...

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
