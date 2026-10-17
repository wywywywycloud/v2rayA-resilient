#!/bin/sh
# Add the signed Resilient feed; install the fork later through LuCI Software.
# Key rotated 2026-09-29: old key 9478c50315c92021 lost, new key 1c624cadb7ee79e7.
set -eu
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run as root on OpenWrt.' >&2; exit 1; }
. /etc/openwrt_release
# APK uses a separate signed index and key; keep the existing opkg feed intact.
add_apk_feed() {
    command -v apk >/dev/null 2>&1 || { echo 'OpenWrt 25.12 requires apk.' >&2; exit 1; }
    arch=$(cat /etc/apk/arch)
    [ "$arch" = "$DISTRIB_ARCH" ] || { echo "APK architecture does not match OpenWrt." >&2; exit 1; }
    case "$arch" in
        aarch64_cortex-a53|aarch64_generic) ;;
        *) echo "No xray-proxy-client-experimental APK build for architecture: $arch" >&2; exit 1 ;;
    esac
    feed_url="https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/openwrt-feed/openwrt-25.12/resilient/$arch"
    key_url='https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/openwrt-feed/luca-apk.pem.pub'
    key_sha256='c1800dbd25023c5b2629d13d02103c621b2ce23e48282f949b88d9a81dda7277'
    work_dir=$(mktemp -d /tmp/luca-apk.XXXXXX)
    trap 'rm -rf "$work_dir"' EXIT
    trap 'exit 1' HUP INT TERM
    mkdir "$work_dir/keys"
    wget -O "$work_dir/keys/luca-apk.pem.pub" "$key_url"
    printf '%s  %s\n' "$key_sha256" "$work_dir/keys/luca-apk.pem.pub" | sha256sum -c -
    wget -O "$work_dir/packages.adb" "$feed_url/packages.adb"
    apk --keys-dir "$work_dir/keys" verify "$work_dir/packages.adb"
    mkdir -p /etc/apk/keys /etc/apk/repositories.d
    cp "$work_dir/keys/luca-apk.pem.pub" /etc/apk/keys/luca-apk.pem.pub
    chmod 644 /etc/apk/keys/luca-apk.pem.pub
    repo_file=/etc/apk/repositories.d/luca-xray-key-bot.list
    printf '%s/packages.adb\n' "$feed_url" > "$work_dir/repository"
    if ! cmp -s "$work_dir/repository" "$repo_file"; then
        [ ! -f "$repo_file" ] || cp "$repo_file" "$repo_file.before-luca"
        cp "$work_dir/repository" "$repo_file.new"
        chmod 644 "$repo_file.new"
        mv "$repo_file.new" "$repo_file"
    fi
    apk update
    echo 'Feed added. Install with: apk add xray-proxy-client-experimental'
    echo 'In LuCI: System > Software > filter xray-proxy-client-experimental > Install.'
    echo 'No application package or application setting was changed by this script.'
}
case "$DISTRIB_RELEASE" in
    25.12.*) add_apk_feed; exit 0 ;;
esac
case "$DISTRIB_RELEASE" in
    24.10.0|24.10.1|24.10.2|24.10.3|24.10.4|24.10.5|24.10.6|24.10.7|24.10.8) ;;
    *) echo 'This feed supports OpenWrt 24.10.0 through 24.10.8 (opkg) and 25.12 (apk).' >&2; exit 1 ;;
esac
opkg print-architecture | awk '$2 == "aarch64_cortex-a53" {found=1} END {exit !found}' || {
    echo 'Requires aarch64_cortex-a53; do not override your router architecture.' >&2; exit 1;
}
feed_url='https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/openwrt-feed/openwrt-24.10/resilient/aarch64_cortex-a53'
# Pin the key to the immutable publication commit to avoid negative CDN caching.
key_url='https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/27fb3a9a7d781d6fe0c7d60a6ffff2113d6ec19b/resilient-key.pub'
key_sha256='0f4cd283f4885d7c32b13c4a2451e6acb9101d0b2324256bc3dad043a81edc9d'
key_id='1c624cadb7ee79e7'
old_key_id='9478c50315c92021'
work_dir=$(mktemp -d /tmp/v2raya-resilient.XXXXXX)
trap 'rm -rf "$work_dir"' EXIT
trap 'exit 1' HUP INT TERM
wget -O "$work_dir/key.pub" "$key_url"
printf '%s  %s\n' "$key_sha256" "$work_dir/key.pub" | sha256sum -c -
[ "$(usign -F -p "$work_dir/key.pub")" = "$key_id" ]
wget -O "$work_dir/Packages" "$feed_url/Packages"
wget -O "$work_dir/Packages.sig" "$feed_url/Packages.sig"
usign -V -m "$work_dir/Packages" -p "$work_dir/key.pub" -x "$work_dir/Packages.sig"
mkdir -p /etc/opkg/keys
cp "$work_dir/key.pub" "/etc/opkg/keys/$key_id"
chmod 644 "/etc/opkg/keys/$key_id"
# Drop the lost rotated-out key so a stale signature cannot be trusted.
rm -f "/etc/opkg/keys/$old_key_id"
touch /etc/opkg/customfeeds.conf
cp /etc/opkg/customfeeds.conf "$work_dir/customfeeds.before"
# Replace only this project's previous sources; preserve all unrelated feeds.
awk '$2 != "v2raya_resilient" && $2 != "v2raya_levin" && $2 != "v2raya_current" && $2 != "v2raya_fork"' /etc/opkg/customfeeds.conf > "$work_dir/customfeeds.conf"
printf 'src/gz v2raya_resilient %s\n' "$feed_url" >> "$work_dir/customfeeds.conf"
if ! cmp -s "$work_dir/customfeeds.before" "$work_dir/customfeeds.conf"; then
    cp "$work_dir/customfeeds.before" /etc/opkg/customfeeds.conf.before-resilient
    cat "$work_dir/customfeeds.conf" > /etc/opkg/customfeeds.conf
fi
rm -f /var/opkg-lists/v2raya_levin /var/opkg-lists/v2raya_levin.sig /var/opkg-lists/v2raya_current /var/opkg-lists/v2raya_fork
rm -f /var/opkg-lists/v2raya_resilient /var/opkg-lists/v2raya_resilient.sig
opkg update
echo 'Feed added. In LuCI: System > Software > filter xray-proxy-client-experimental > Install xray-proxy-client-experimental.'
echo 'No application package or application setting was changed by this script.'
