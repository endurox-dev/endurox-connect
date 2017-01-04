# Do recursive builds
all:
	go get github.com/endurox-dev/endurox-go
	$(MAKE) -C ubftab
	$(MAKE) -C restincl
	$(MAKE) -C tcpgatesv

clean:
	$(MAKE) -C ubftab clean
	$(MAKE) -C restincl clean
	$(MAKE) -C tcpgatesv clean
