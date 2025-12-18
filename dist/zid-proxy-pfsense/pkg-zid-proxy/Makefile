# Created by: ZID Proxy Team
# $FreeBSD$

PORTNAME=	pfSense-pkg-zid-proxy
PORTVERSION=	1.0.0
CATEGORIES=	net-mgmt
MASTER_SITES=	# empty
DISTFILES=	# empty
EXTRACT_ONLY=	# empty

MAINTAINER=	guilherme@example.com
COMMENT=	pfSense package zid-proxy - SNI-based transparent HTTPS proxy

LICENSE=	APACHE20

NO_BUILD=	yes
NO_MTREE=	yes

SUB_FILES=	pkg-install pkg-deinstall
SUB_LIST=	PORTNAME=${PORTNAME}

do-extract:
	${MKDIR} ${WRKSRC}

do-install:
	# Create directories
	${MKDIR} ${STAGEDIR}${PREFIX}/pkg
	${MKDIR} ${STAGEDIR}${PREFIX}/www
	${MKDIR} ${STAGEDIR}${PREFIX}/etc/rc.d
	${MKDIR} ${STAGEDIR}${PREFIX}/sbin
	${MKDIR} ${STAGEDIR}${PREFIX}/etc/zid-proxy
	${MKDIR} ${STAGEDIR}/etc/inc/priv
	${MKDIR} ${STAGEDIR}${DATADIR}

	# Install package configuration files
	${INSTALL_DATA} ${FILESDIR}${PREFIX}/pkg/zid-proxy.xml \
		${STAGEDIR}${PREFIX}/pkg
	${INSTALL_DATA} ${FILESDIR}${PREFIX}/pkg/zid-proxy.inc \
		${STAGEDIR}${PREFIX}/pkg

	# Install web pages
	${INSTALL_DATA} ${FILESDIR}${PREFIX}/www/zid-proxy_rules.php \
		${STAGEDIR}${PREFIX}/www
	${INSTALL_DATA} ${FILESDIR}${PREFIX}/www/zid-proxy_log.php \
		${STAGEDIR}${PREFIX}/www

	# Install privilege definitions
	${INSTALL_DATA} ${FILESDIR}/etc/inc/priv/zid-proxy.priv.inc \
		${STAGEDIR}/etc/inc/priv

	# Install package info
	${INSTALL_DATA} ${FILESDIR}${DATADIR}/info.xml \
		${STAGEDIR}${DATADIR}

	# Note: The binary (zid-proxy) and rc.d script should be installed
	# separately or included in a dependency package

.include <bsd.port.mk>
