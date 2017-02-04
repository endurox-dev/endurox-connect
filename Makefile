

.PHONY: all pkg


# Do recursive builds
all:
	$(MAKE) -C tests
	$(MAKE) -C go
	cd pkg && cmake .

doc:
	$(MAKE) -C doc
	
pkg: all
	cd pkg && cpack

