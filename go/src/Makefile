# Do recursive builds
all:
	go get -u github.com/endurox-dev/endurox-go
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
