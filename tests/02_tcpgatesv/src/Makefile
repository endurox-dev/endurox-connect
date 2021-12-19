# Do recursive builds
all:
	go get -u github.com/endurox-dev/endurox-go && cd github.com/endurox-dev/endurox-go && git checkout v8.0
	$(MAKE) -C ubftab
	$(MAKE) -C testsv
	$(MAKE) -C testcl

clean:
	$(MAKE) -C ubftab clean
	$(MAKE) -C testsv clean
	$(MAKE) -C testcl clean

.PHONY: clean all
