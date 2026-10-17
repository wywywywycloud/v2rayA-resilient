# OpenWrt 25.12 APK feed

Install the complete xray-proxy-client-experimental client:

    wget -O /tmp/add-resilient-feed.sh https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/openwrt-feed/add-resilient-feed.sh &&
    sh /tmp/add-resilient-feed.sh &&
    apk add xray-proxy-client-experimental

The bootstrap checks /etc/apk/arch and installs the correct repository for aarch64_cortex-a53 or aarch64_generic. It verifies the public key hash and signed packages.adb before registering the feed. No --allow-untrusted flag is needed.

The bundle contains a service, matching core and LuCI integration. Exact version dependencies prevent mixing releases. Existing internal configuration and service names remain v2raya; the application still displays x-Ray VPN. Open the web interface on port 2017 or use LuCI Services > x-Ray VPN.

Application base: b51db60a6654b50372bfe6d4f3d63a204b928fc5, identical to the source of the current 2026.09.29-experimental.1 IPK release. APK build sources and tests are maintained on main-apk-feed in wywywywycloud/v2rayA-current. See the release validation for the tested OpenWrt version and limitations.

The existing OpenWrt 24.10 opkg feed and package names remain unchanged.

Release name: luca-xray-key-bot. To migrate an already installed initial APK, run `apk del luca-xray-key-bot` before the installation above; modified settings are preserved.
