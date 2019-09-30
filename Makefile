

.PHONY: all pkg clean doc


# Do recursive builds
all:
	$(MAKE) -C go
	$(MAKE) -C tests
	cd pkg && cmake .

clean:
	$(MAKE) -C tests clean
	$(MAKE) -C go clean

doc:
	$(MAKE) -C doc
	
pkg: all
	cd pkg && cpack

