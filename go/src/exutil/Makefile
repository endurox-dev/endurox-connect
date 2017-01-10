
SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

LIBRARY=exutil
LDFLAGS=

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

.DEFAULT_GOAL: $(LIBRARY)

$(LIBRARY): $(SOURCES)
	go build ${LDFLAGS} -o ${LIBRARY} *.go
	go install ${LDFLAGS} ./...

.PHONY: clean
clean:
	if [ -f ${LIBRARY} ] ; then rm ${LIBRARY} ; fi

