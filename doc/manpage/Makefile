##
## @brief Enduro/X Addons (connectivity) module documentation
##
## @file Makefile
##
## -----------------------------------------------------------------------------
## Enduro/X Middleware Platform for Distributed Transaction Processing
## Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
## Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
## This software is released under one of the following licenses:
## AGPL or Mavimax's license for commercial use.
## -----------------------------------------------------------------------------
## AGPL license:
## 
## This program is free software; you can redistribute it and/or modify it under
## the terms of the GNU Affero General Public License, version 3 as published
## by the Free Software Foundation;
##
## This program is distributed in the hope that it will be useful, but WITHOUT ANY
## WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
## PARTICULAR PURPOSE. See the GNU Affero General Public License, version 3
## for more details.
##
## You should have received a copy of the GNU Affero General Public License along 
## with this program; if not, write to the Free Software Foundation, Inc., 
## 59 Temple Place, Suite 330, Boston, MA 02111-1307 USA
##
## -----------------------------------------------------------------------------
## A commercial use license is available from Mavimax, Ltd
## contact@mavimax.com
## -----------------------------------------------------------------------------
##

A2X := $(shell a2x --version 2> /dev/null)

SOURCE_DOCS :=$(shell ls -1 *.adoc)
DOCS  := $(patsubst %.adoc, %.8, $(SOURCE_DOCS))

.DEFAULT_GOAL := all

check_asciidoc: 
	@type a2x >/dev/null 2>&1 || echo "Missing asciidoc - docs will not built"

all: check_asciidoc $(DOCS)

#
# Do not remove images
#
.SECONDARY: $(GRAPHS)


#
# Build the document
# So translate the % document and use $%
#
%.8: %.adoc
	a2x -D. --format=manpage $<
	
clean:
	rm -f *.8

.PHONY: clean all



# vim: set ts=4 sw=4 et smartindent:
