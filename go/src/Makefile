# Do recursive builds
all:
	go get -u github.com/endurox-dev/endurox-go && cd github.com/endurox-dev/endurox-go && git checkout v8.0
	$(MAKE) -C ubftab
	$(MAKE) -C exutil
	$(MAKE) -C restincl
	$(MAKE) -C restoutsv
	$(MAKE) -C tcpgatesv

clean:
	- rm -rf github.com/endurox-dev
	$(MAKE) -C ubftab clean
	$(MAKE) -C exutil clean
	$(MAKE) -C restincl clean
	$(MAKE) -C restoutsv clean
	$(MAKE) -C tcpgatesv clean

.PHONY: clean
