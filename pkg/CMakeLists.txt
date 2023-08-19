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

cmake_minimum_required (VERSION 3.1) 
project (ENDUROX-CONNECT) 
set(VERSION "8.0.4")
set(PROJ_NAME "Enduro/X Connectivity Module")
set(RELEASE "1")

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "/usr/share/endurox/cmake")
include(ex_osver)

################################################################################
# Process any required includes
################################################################################
ex_osver_include()
################################################################################


################################################################################
# Check versions if defined, export build env
################################################################################

if(DEFINED ENV{NDRX_BLD_VERSION})
        IF(NOT $ENV{NDRX_BLD_VERSION} MATCHES ${VERSION})
                message( FATAL_ERROR "Invalid versions: ENV: [$ENV{NDRX_BLD_VERSION}] Code: [${VERSION}]" )
        endif()
endif()

if(DEFINED ENV{NDRX_BLD_RELEASE})
    set(RELEASE $ENV{NDRX_BLD_RELEASE})
endif()

message("CMake RELEASE  = ${RELEASE}")

SET (NDRX_BLD_PRODUCT   $ENV{NDRX_BLD_PRODUCT})
SET (NDRX_BLD_SYSTEM    $ENV{NDRX_BLD_SYSTEM})
SET (NDRX_BLD_CONFIG    $ENV{NDRX_BLD_CONFIG})
SET (NDRX_BLD_VERSION   $ENV{NDRX_BLD_VERSION})
SET (NDRX_BLD_RELEASE   $ENV{NDRX_BLD_RELEASE})
SET (NDRX_BLD_TAG       $ENV{NDRX_BLD_TAG})
SET (NDRX_BLD_BRANCH    $ENV{NDRX_BLD_BRANCH})
SET (NDRX_BLD_COMMIT    $ENV{NDRX_BLD_COMMIT})
SET (NDRX_BLD_FLAGS     $ENV{NDRX_BLD_FLAGS})


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
	set ( CMAKE_C_FLAGS "-D_SEM_SEMUN_UNDEFINED -D_MSGQSUPPORT -D_THREAD_SAFE -pthread -maix64 -Wl,-brtl -Wl,-G ${CMAKE_C_FLAGS}")
	set ( CMAKE_CXX_FLAGS "-D_SEM_SEMUN_UNDEFINED -D_MSGQSUPPORT -D_THREAD_SAFE -pthread -maix64 -Wl,-brtl -Wl,-G ${CMAKE_CXX_FLAGS}")
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
    
# check that a2x actually works, on aix there is a2x but it is not asciidoc

execute_process(COMMAND a2x "--version" RESULT_VARIABLE RET)
if(A2X_EXECUTABLE)

    if( NOT ${RET} MATCHES "0")
        message("a2x does not work - disabling doc...")
        unset(A2X_EXECUTABLE)
    elseif()
	# Install manpages (if any
	install (FILES
		../doc/manpage/restincl.8
		../doc/manpage/restousv.8
		../doc/manpage/tcpgatesv.8
		DESTINATION share/man/man8)

	install (FILES
		../doc/manpage/tcpgatesv.html
		../doc/manpage/restincl.html
		../doc/manpage/restoutsv.html
		DESTINATION share/endurox-connect/doc/html/manpage)

	install (FILES
  		../doc/manpage/docbook-xsl.css
		DESTINATION share/endurox-connect/doc/html/manpage
  		OPTIONAL)

	install (FILES
                ../doc/restinout_user_guide.html
		../doc/tcpgatesv_user_guide.html
		../doc/nonpers_sync_ex_to_net.png
		../doc/nonpers_sync_net_to_ex.png
		../doc/pers_async_ex_to_net_wo_corr.png
		../doc/pers_async_net_to_ex_w_corr.png
		../doc/pers_async_net_to_ex_wo_corr.png
		../doc/pers_sync_ex_to_net_connid.png
		../doc/pers_sync_net_to_ex.png
		../doc/restinout_tutor_1.png
                DESTINATION share/endurox-connect/doc/html/guides)

	install (FILES
  		../doc/docbook-xsl.css
		DESTINATION share/endurox-connect/doc/html/guides
  		OPTIONAL)
    endif()
endif()
	
################################################################################
# Packages
################################################################################
ex_osver()
ex_cpuarch()

MESSAGE( "CPack:Debug: CMAKE_SYSTEM_NAME = " ${CMAKE_SYSTEM_NAME} )
MESSAGE( "CPack:Debug: CMAKE_SYSTEM_PROCESSOR = " ${CMAKE_SYSTEM_PROCESSOR} )
set(CPACK_MONOLITHIC_INSTALL 1)

find_program(RPMPROG "rpmbuild")
find_program(APTPROG "dpkg")

# avoid file /usr/share/man from install of endurox-3.5.1-1.x86_64 conflicts with file from package filesystem-3.2-21.el7.x86_64
# problems...
set(CPACK_RPM_EXCLUDE_FROM_AUTO_FILELIST_ADDITION /usr/share/man
		/usr/share/man/man3
		/usr/share/man/man5
		/usr/share/man/man8
		/usr/share/java
		/usr/lib64/pkgconfig
		/usr/lib/pkgconfig)
message("Excl: ${CPACK_RPM_EXCLUDE_FROM_AUTO_FILELIST_ADDITION}")

set(CPACK_GENERATOR "TGZ")
if(RPMPROG)
        message("Outputting RPM")
	set(CPACK_GENERATOR "${CPACK_GENERATOR};RPM")
endif()

if(APTPROG)
        message("Outputting DEB")
	set(CPACK_GENERATOR "${CPACK_GENERATOR};DEB")
endif()

INCLUDE (${CMAKE_ROOT}/Modules/CheckTypeSize.cmake)
CHECK_TYPE_SIZE("void*"  EX_SIZEOF_VOIDPTR)
MATH (EXPR EX_PLATFORM_BITS "${EX_SIZEOF_VOIDPTR} * 8")

message("Generator: ${CPACK_GENERATOR}")

set(CPACK_PACKAGE_CONTACT "contact@mavimax.com")
set(CPACK_PACKAGE_VERSION ${VERSION})
set(CPACK_PACKAGE_VENDOR "Mavimax Ltd")
#set(CPACK_DEBIAN_PACKAGE_DEPENDS "endurox")
#set(CPACK_RPM_PACKAGE_REQUIRES "endurox")
string(TOLOWER ${PROJECT_NAME} PROJECT_NAME_LO)
message("CPack:Debug: PROJECT NAME = ${PROJECT_NAME_LO}")
set(CPACK_PACKAGE_FILE_NAME         ${PROJECT_NAME_LO}-${VERSION}-${RELEASE}.${LSB_RELEASE_OUTPUT_OS}${LSB_RELEASE_OUTPUT_VER}.${EX_CPU_ARCH})
string(TOLOWER ${CPACK_PACKAGE_FILE_NAME}  CPACK_PACKAGE_FILE_NAME)
message("CPack:Debug: CPACK_PACKAGE_FILE_NAME = ${CPACK_PACKAGE_FILE_NAME}")
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Enduro/X Connectivity Module")
set(CPACK_DEBIAN_PACKAGE_DESCRIPTION  ${CPACK_PACKAGE_DESCRIPTION})
set(CPACK_RPM_PACKAGE_DESCRIPTION ${CPACK_PACKAGE_DESCRIPTION})
set(CPACK_RPM_PACKAGE_AUTOREQ "0")
#set(CPACK_DEBIAN_PACKAGE_CONTROL_EXTRA "${CMAKE_CURRENT_SOURCE_DIR}/Debian/postinst")

include(CPack)
################################################################################


# vim: set ts=4 sw=4 et smartindent:
