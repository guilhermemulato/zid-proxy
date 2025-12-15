<?php
/*
 * zid-proxy.priv.inc
 *
 * Privilege definitions for ZID Proxy pfSense package
 *
 * Licensed under the Apache License, Version 2.0
 */

global $priv_list;

$priv_list['page-services-zidproxy'] = array();
$priv_list['page-services-zidproxy']['name'] = "WebCfg - Services: ZID Proxy";
$priv_list['page-services-zidproxy']['descr'] = "Allow access to ZID Proxy package settings";
$priv_list['page-services-zidproxy']['match'] = array();
$priv_list['page-services-zidproxy']['match'][] = "pkg.php?xml=zid-proxy.xml*";
$priv_list['page-services-zidproxy']['match'][] = "pkg_edit.php?xml=zid-proxy.xml*";
$priv_list['page-services-zidproxy']['match'][] = "zid-proxy_rules.php*";
$priv_list['page-services-zidproxy']['match'][] = "zid-proxy_log.php*";

?>
