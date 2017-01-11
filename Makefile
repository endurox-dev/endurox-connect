

.PHONY: all pkg


# Do recursive builds
all:
	$(MAKE) -C tests
	$(MAKE) -C go
	$(MAKE) -C doc
	cd pkg; cmake .

pkg: all
	cd pkg; cpack

