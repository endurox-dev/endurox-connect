##
## @brief Connectivity packaging scripts
##
## @file CMakeLists.txt
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

cmake_minimum_required (VERSION 2.6) 
project (ENDUROXCONNECT) 
set(VERSION "2.4.12")
set(PROJ_NAME "Enduro/X Connectivity Module")
set(RELEASE "1")

################################################################################
# OS Configuration
################################################################################

find_program(A2X_EXECUTABLE NAMES a2x)


EXEC_PROGRAM(uname ARGS -a OUTPUT_VARIABLE _TMP_CMAKE_OS_NAME)
string(REGEX MATCH "^[a-zA-Z0-9-]*" _TMP_CMAKE_OS_NAME_EXTR ${_TMP_CMAKE_OS_NAME})
string(TOUPPER ${_TMP_CMAKE_OS_NAME_EXTR} CMAKE_OS_NAME)

message("CMake CMAKE_OS_NAME  = ${CMAKE_OS_NAME}")
message("CMake CMAKE_C_COMPILER_ID  = ${CMAKE_C_COMPILER_ID}")
message("CMake CMAKE_CXX_COMPILER_ID  = ${CMAKE_CXX_COMPILER_ID}")

if( ${CMAKE_OS_NAME} STREQUAL "LINUX" )
        set(EX_OS_LINUX "1")
elseif(${CMAKE_OS_NAME} STREQUAL "AIX")
	set(EX_OS_AIX "1")
elseif(${CMAKE_OS_NAME} STREQUAL "HP-UX")
	set(EX_OS_HPUX "1")
elseif(${CMAKE_OS_NAME} STREQUAL "SUNOS")
	set(EX_OS_SUNOS "1")
elseif(${CMAKE_OS_NAME} STREQUAL "FREEBSD")
	set(EX_OS_FREEBSD "1")
elseif(${CMAKE_OS_NAME} STREQUAL "CYGWIN")
	set(EX_OS_CYGWIN "1")
elseif(${CMAKE_OS_NAME} STREQUAL "DARWIN")
	set(EX_OS_DARWIN "1")
# Unkonwn OS:
else()
	message( FATAL_ERROR "Unsupported OS" )
endif()


if(${CMAKE_OS_NAME} STREQUAL "DARWIN")
    set(LIB_SUFFIX "")
elseif ("${LIB64}" STREQUAL "TRUE")
    set(LIB_SUFFX 64)
else()
    set(LIB_SUFFIX "")
endif()

set(INSTALL_LIB_DIR     lib${LIB_SUFFIX} CACHE PATH "Installation directory for libraries")
mark_as_advanced(INSTALL_LIB_DIR) 
MESSAGE( STATUS "INSTALL_LIB_DIR: " ${INSTALL_LIB_DIR} )
################################################################################
# Files to install
################################################################################

install (FILES
    ../go/src/tcpgatesv/tcpgatesv
    ../go/src/restincl/restincl
    ../go/src/restoutsv/restoutsv
    PERMISSIONS OWNER_EXECUTE OWNER_WRITE OWNER_READ GROUP_EXECUTE GROUP_READ WORLD_EXECUTE WORLD_READ
    DESTINATION bin)
    
if(A2X_EXECUTABLE)
	# Install manpages (if any
	install (FILES
		../doc/manpage/restincl.8
		../doc/manpage/tcpgatesv.8
		DESTINATION share/man/man8)
endif()
	
################################################################################
# Packages
################################################################################

find_program(LSB_RELEASE_EXECUTABLE lsb_release)
if(LSB_RELEASE_EXECUTABLE)
        execute_process(COMMAND ${LSB_RELEASE_EXECUTABLE} -si
                OUTPUT_VARIABLE _TMP_LSB_RELEASE_OUTPUT_OS
                ERROR_QUIET
                OUTPUT_STRIP_TRAILING_WHITESPACE)

        string(TOLOWER 
                ${_TMP_LSB_RELEASE_OUTPUT_OS}
        LSB_RELEASE_OUTPUT_OS)
        string(REPLACE " " "_" LSB_RELEASE_OUTPUT_OS ${LSB_RELEASE_OUTPUT_OS})

        execute_process(COMMAND ${LSB_RELEASE_EXECUTABLE} -sr
                OUTPUT_VARIABLE _TMP_LSB_RELEASE_OUTPUT_VER
                ERROR_QUIET
                OUTPUT_STRIP_TRAILING_WHITESPACE)
        string(REPLACE "." "_" LSB_RELEASE_OUTPUT_VER ${_TMP_LSB_RELEASE_OUTPUT_VER})
        #string(REGEX MATCH "^[0-9]+" LSB_RELEASE_OUTPUT_VER ${_TMP_LSB_RELEASE_OUTPUT_VER})
elseif (EXISTS /etc/os-release)

	execute_process(COMMAND bash -c "cat /etc/os-release | egrep '^NAME=' | cut -d '\"' -f2"
                OUTPUT_VARIABLE _TMP_LSB_RELEASE_OUTPUT_OS
                OUTPUT_STRIP_TRAILING_WHITESPACE)

        string(TOLOWER 
                ${_TMP_LSB_RELEASE_OUTPUT_OS}
        LSB_RELEASE_OUTPUT_OS)
        string(REPLACE " " "_" LSB_RELEASE_OUTPUT_OS ${LSB_RELEASE_OUTPUT_OS})

        execute_process(COMMAND bash -c "cat /etc/os-release | egrep '^VERSION=' | cut -d '\"' -f2"
                OUTPUT_VARIABLE _TMP_LSB_RELEASE_OUTPUT_VER
                OUTPUT_STRIP_TRAILING_WHITESPACE)
        string(REPLACE "." "_" LSB_RELEASE_OUTPUT_VER ${_TMP_LSB_RELEASE_OUTPUT_VER})
        #string(REGEX MATCH "^[0-9]+" LSB_RELEASE_OUTPUT_VER ${_TMP_LSB_RELEASE_OUTPUT_VER})
else ()
        #set(LSB_RELEASE_OUTPUT_OS "unknown")
        #set(LSB_RELEASE_OUTPUT_VER "0")
	set(LSB_RELEASE_OUTPUT_OS ${CMAKE_OS_NAME})
	string(REPLACE "." "_" _TMP_LSB_RELEASE_OUTPUT_VER ${CMAKE_SYSTEM_VERSION})
	string(REPLACE "-" "_" LSB_RELEASE_OUTPUT_VER ${_TMP_LSB_RELEASE_OUTPUT_VER})
endif()

message("CPack:Debug: LSB_RELEASE OS  = ${LSB_RELEASE_OUTPUT_OS}")
message("CPack:Debug: LSB_RELEASE VER = ${LSB_RELEASE_OUTPUT_VER}")


# Support only for Linux
MESSAGE( "CPack:Debug: CMAKE_SYSTEM_NAME: " ${CMAKE_SYSTEM_NAME} )
MESSAGE( "CPack:Debug: CMAKE_SYSTEM_PROCESSOR: " ${CMAKE_SYSTEM_PROCESSOR} )

find_program(RPMPROG "rpmbuild")
find_program(APTPROG "apt")

set(CPACK_GENERATOR "TGZ")
if(RPMPROG)
        message("Outputting RPM")
        set(CPACK_GENERATOR "${CPACK_GENERATOR};RPM")
endif()

if(APTPROG)
        message("Outputting DEB")
        set(CPACK_GENERATOR "${CPACK_GENERATOR};DEB")
endif()


# have some issues with folders
set(CPACK_RPM_EXCLUDE_FROM_AUTO_FILELIST_ADDITION /usr/share/man
		/usr/share/man/man3
		/usr/share/man/man5
		/usr/share/man/man8
		/usr/lib64/pkgconfig
		/usr/lib/pkgconfig)

set(CPACK_MONOLITHIC_INSTALL 1)
set(CPACK_PACKAGE_CONTACT "madars.vitolins@gmail.com")
set(CPACK_PACKAGE_VERSION ${VERSION})
#set(CPACK_DEBIAN_PACKAGE_DEPENDS "endurox")
#set(CPACK_RPM_PACKAGE_REQUIRES "endurox")
string(TOLOWER ${PROJECT_NAME} PROJECT_NAME_LO)
message("CPack:Debug: PROJECT NAME = ${PROJECT_NAME_LO}")
set(CPACK_PACKAGE_FILE_NAME         ${PROJECT_NAME_LO}-${VERSION}-${RELEASE}.${LSB_RELEASE_OUTPUT_OS}${LSB_RELEASE_OUTPUT_VER}.${CMAKE_SYSTEM_PROCESSOR})
message("CPack:Debug: CPACK_PACKAGE_FILE_NAME = ${CPACK_PACKAGE_FILE_NAME}")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Enduro/X Connectivity Module")
set(CPACK_DEBIAN_PACKAGE_DESCRIPTION  ${CPACK_PACKAGE_DESCRIPTION})
set(CPACK_RPM_PACKAGE_DESCRIPTION ${CPACK_PACKAGE_DESCRIPTION})
#set(CPACK_DEBIAN_PACKAGE_CONTROL_EXTRA "${CMAKE_CURRENT_SOURCE_DIR}/Debian/postinst")

include(CPack)
################################################################################



# vim: set ts=4 sw=4 et smartindent:
