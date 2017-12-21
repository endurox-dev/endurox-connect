# Do recursive builds
all:
	go get -u github.com/endurox-dev/endurox-go
	$(MAKE) -C ubftab
	$(MAKE) -C testsv
	$(MAKE) -C testcl

clean:
	$(MAKE) -C ubftab clean
	$(MAKE) -C testsv clean
	$(MAKE) -C testcl clean
	cd github.com/endurox-dev/endurox-go && git pull
