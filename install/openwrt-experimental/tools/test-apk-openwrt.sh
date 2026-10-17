#!/bin/sh
# Destructive package lifecycle checks for an explicitly disposable VM only.
set -eu
[ "${V2RAYA_DISPOSABLE_VM:-}" = 1 ] || { echo 'Disposable VM required' >&2; exit 1; }
. /etc/openwrt_release
[ "$DISTRIB_RELEASE" = 25.12.5 ]
[ "$(uname -m)" = aarch64 ]
out=${1:-/usr/luca-tests/evidence/packages}
mkdir -p "$out"
exec > "$out/test.log" 2>&1
version=2026.10.17-luca.1
wait_api() {
    n=0
    until wget -qO "$out/version.json" http://127.0.0.1:2017/api/version &&
          grep -q "$version" "$out/version.json"; do
        n=$((n + 1))
        [ "$n" -lt 90 ]
        sleep 1
    done
    /etc/init.d/v2raya running
}
apk add xray-proxy-client-experimental
for package in xray-proxy-client-experimental xray-proxy-client-experimental-core xray-proxy-client-experimental-service luci-app-xray-proxy-client-experimental; do
    apk info -e "$package"
done
sha256sum /usr/bin/v2raya /usr/bin/v2raya_core > "$out/binaries.sha256"
[ "$(ls -ln /usr/bin/v2raya | awk '{print $3}')" = 0 ]
[ "$(ls -ln /usr/bin/v2raya_core | awk '{print $3}')" = 0 ]
[ -f /usr/share/luci/menu.d/luci-app-v2raya.json ]
[ -f /www/luci-static/resources/view/v2raya/config.js ]
[ "$(uci -q get v2raya.config.enabled)" = 1 ]
/etc/init.d/v2raya enabled
wait_api
wget -qO "$out/index.html" http://127.0.0.1:2017/
grep -q '<html' "$out/index.html"
before=$(pidof v2raya)
/etc/init.d/v2raya restart
sleep 2
wait_api
[ "$before" != "$(pidof v2raya)" ]
uci set v2raya.config.log_level=warn
uci commit v2raya
apk fix xray-proxy-client-experimental-service
wait_api
[ "$(uci -q get v2raya.config.log_level)" = warn ]
sha256sum -c "$out/binaries.sha256"
before=$(pidof v2raya)
kill -9 "$before"
sleep 18
wait_api
[ "$before" != "$(pidof v2raya)" ]
apk del xray-proxy-client-experimental
! pidof v2raya
[ ! -e /etc/rc.d/S99v2raya ]
apk add xray-proxy-client-experimental
wait_api
[ "$(uci -q get v2raya.config.log_level)" = warn ]
sha256sum -c "$out/binaries.sha256"
uci set v2raya.config.log_level=info
uci commit v2raya
/etc/init.d/v2raya restart
sleep 2
wait_api
ubus call service list '{"name":"v2raya"}' > "$out/procd.json"
apk info > "$out/installed.txt"
echo PASS > "$out/result"
