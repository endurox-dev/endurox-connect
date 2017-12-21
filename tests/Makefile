# Do recursive builds
all:
	$(MAKE) -C 01_restin
	$(MAKE) -C 02_tcpgatesv
	$(MAKE) -C 03_restout

clean:
	$(MAKE) -C 01_restin clean
	$(MAKE) -C 02_tcpgatesv clean
	$(MAKE) -C 03_restout clean

.PHONY: clean all
