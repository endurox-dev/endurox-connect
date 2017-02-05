

.PHONY: all pkg clean


# Do recursive builds
all:
	$(MAKE) -C tests
	$(MAKE) -C go
	cd pkg && cmake .

clean:
	$(MAKE) -C tests clean
	$(MAKE) -C go clean

doc:
	$(MAKE) -C doc
	
pkg: all
	cd pkg && cpack

