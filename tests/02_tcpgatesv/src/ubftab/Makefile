
SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.fd')
SOURCES += Exfields

OUTPUT = $(addsuffix .go, $(SOURCES))

comma:= ,
empty:=
space:= $(empty) $(empty)

export FIELDTBLS=$(subst $(space),$(comma),$(SOURCES))
export FLDTBLDIR=$(shell pwd)

$(info $$SOURCES is [${SOURCES}])
$(info $$OUTPUT is [${OUTPUT}])
$(info $$FIELDTBLS is [${FIELDTBLS}])

PACKAGE=ubftab

.DEFAULT_GOAL: $(OUTPUT)
LDFLAGS=

$(OUTPUT): $(SOURCES)
	mkfldhdr -m1 -p${PACKAGE}
	go build ${LDFLAGS} -o ${PACKAGE} *.go
	go install ${LDFLAGS} ./...

.PHONY: clean
clean:
	- rm *.go
