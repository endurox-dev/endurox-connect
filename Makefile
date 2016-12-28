# Do recursive builds
all:
	$(MAKE) -C tests
	$(MAKE) -C go
