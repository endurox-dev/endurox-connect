all:
	go get -u github.com/endurox-dev/endurox-go
	$(MAKE) -C ubftab
	$(MAKE) -C testcl
	$(MAKE) -C viewdir
	$(MAKE) -C bigmsgsv

clean:
	$(MAKE) -C ubftab clean
	$(MAKE) -C testcl clean
	$(MAKE) -C viewdir clean
	$(MAKE) -C bigmsgsv clean


.PHONY: clean all

