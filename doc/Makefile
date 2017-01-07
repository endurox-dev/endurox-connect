## 
## Enduro/X Addons (connectivity) module documentation
##
## @file Makefile
## 
## -----------------------------------------------------------------------------
## Enduro/X Middleware Platform for Distributed Transaction Processing
## Copyright (C) 2015, ATR Baltic, SIA. All Rights Reserved.
## This software is released under one of the following licenses:
## GPL or ATR Baltic's license for commercial use.
## -----------------------------------------------------------------------------
## GPL license:
## 
## This program is free software; you can redistribute it and/or modify it under
## the terms of the GNU General Public License as published by the Free Software
## Foundation; either version 2 of the License, or (at your option) any later
## version.
##
## This program is distributed in the hope that it will be useful, but WITHOUT ANY
## WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
## PARTICULAR PURPOSE. See the GNU General Public License for more details.
##
## You should have received a copy of the GNU General Public License along with
## this program; if not, write to the Free Software Foundation, Inc., 59 Temple
## Place, Suite 330, Boston, MA 02111-1307 USA
##
## -----------------------------------------------------------------------------
## A commercial use license is available from ATR Baltic, SIA
## contact@atrbaltic.com
## -----------------------------------------------------------------------------
##

SOURCE_DOCS :=$(shell find -name '*.adoc' -printf "%f\n")
DOCS  := $(patsubst %.adoc, %.html, $(SOURCE_DOCS))

SOURCES_GRAPHS :=$(shell find -name '*.dia' -printf "%f\n")
GRAPHS  := $(patsubst %.dia, %.png, $(SOURCES_GRAPHS))

$(info $$SOURCES_GRAPHS is [${SOURCES_GRAPHS}])
$(info $$GRAPHS is [${GRAPHS}])

all: $(DOCS)

#
# Do not remove images
#
.SECONDARY: $(GRAPHS)


#
# Build the images
# Images shall go to "images" folder
# and later shall be copied to "out"/images
#
%.png: %.dia
	dia --size=600x --filter=png -e $@ -t png $<

#
# Build the document
# So translate the % document and use $%
#
%.html: %.adoc $(GRAPHS)
	a2x -D. -f xhtml $<
	
clean:
	rm -f *.html *.css *.xml *.png

.PHONY: clean all



