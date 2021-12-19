all:
	go get -u github.com/endurox-dev/endurox-go && cd github.com/endurox-dev/endurox-go && git checkout v8.0
	$(MAKE) -C ubftab
	$(MAKE) -C testsv
	$(MAKE) -C transv
	$(MAKE) -C trancl
	$(MAKE) -C viewdir

clean:
	$(MAKE) -C ubftab clean
	$(MAKE) -C testsv clean
	$(MAKE) -C transv clean
	$(MAKE) -C trancl clean
	$(MAKE) -C viewdir clean


.PHONY: clean all
