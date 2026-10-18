# luca-xray-key-bot release: OpenWrt 25.12 packages

APK 2026.10.18-r5 includes verified HTTPS node probes from source commit bd5d54dd3061bcc4ad612c9c8666dfb7caec8f52 on main-apk-feed. The application base and unchanged core originate from b51db60a6654b50372bfe6d4f3d63a204b928fc5 (2026.09.29-experimental.1 IPK). Dashboard list checks share one isolated core and run at most four HTTP requests concurrently. Start/stop cancels the dashboard check before changing core state. A successful TCP connection alone no longer marks a proxy healthy. Probe timings include the HTTPS request, not ICMP RTT.

Supported package architectures: aarch64_cortex-a53 and aarch64_generic. The latter is used by the official ARM64 QEMU image. Other architectures are not supported by this release.

On OpenWrt 25.12:

    wget -O /tmp/add-resilient-feed.sh https://raw.githubusercontent.com/wywywywycloud/v2rayA-current/openwrt-feed/add-resilient-feed.sh &&
    sh /tmp/add-resilient-feed.sh &&
    apk add --upgrade xray-proxy-client-experimental

The metapackage installs xray-proxy-client-experimental-service, xray-proxy-client-experimental-core and luci-app-xray-proxy-client-experimental with exact matching versions. OpenWrt supplies the remaining dependencies, including LuCI, CA certificates, nft TPROXY, kmod-tun and geodata. The application keeps its x-Ray VPN display name, /etc/config/v2raya configuration, /etc/v2raya database and /etc/init.d/v2raya service. Access its interface on port 2017, or through LuCI Services.

The same bootstrap URL retains its signed opkg feed on OpenWrt 24.10.0–24.10.8. APK uses a separate RSA public key and packages.adb index. The script checks the pinned key hash and index signature before adding the repository; it installs no application package itself. Repeated invocation is idempotent.

## Rebuilding

Build the GUI and static Linux ARM64 binaries using README.md, with service version 2026.10.18-luca.5 and core version 2026.10.18-luca.5. Use Go 1.26.8, Node 24.21.0 and Yarn 1.22.22. Do not change the committed Go replacements or lockfiles.

Build upstream apk-tools 3.0.5 with mkpkg and mkndx enabled. Generate a PEM RSA key and a matching .pub file beside it:

    openssl genrsa -out luca-apk.pem 2048
    chmod 600 luca-apk.pem
    openssl rsa -in luca-apk.pem -pubout -out luca-apk.pem.pub
    sudo python3 install/openwrt-experimental/tools/package-apk.py \
      --service dist/v2raya --core dist/v2raya_core \
      --binary-runner qemu-aarch64 \
      --version 2026.10.18-r5 --arch aarch64_cortex-a53 \
      --apk /absolute/path/to/apk --key /absolute/path/to/luca-apk.pem \
      --output dist/apk/aarch64_cortex-a53

Repeat for aarch64_generic with the same binaries and another output directory. The output directory must not already exist. Signing keys are private build inputs and must never be committed. The builder reuses the existing IPK payload construction; APK lifecycle hooks enable/start the service, restart it on upgrade and stop/disable it on removal. UCI configuration is registered with OpenWrt APK conffiles metadata.

Consult APK-VALIDATION.md in the release for actual results and limitations. Physical-router testing is outside the VM test scope.

## Migration from the initial APK naming

If luca-xray-key-bot is already installed, first run `apk del luca-xray-key-bot`, then rerun the bootstrap above and install `xray-proxy-client-experimental`. Modified UCI configuration and /etc/v2raya data are retained. The new packages conflict with the old payload packages to prevent file overwrites. The release name remains luca-xray-key-bot.

## r4 version correction

Both binaries now report 2026.10.18-luca.5. r3 incorrectly reused a core stamped 2026.10.17-luca.1, which triggered the dashboard compatibility error despite working traffic. Core functionality and dependency pins are unchanged; the core was rebuilt with the correct release version. The APK builder now executes both version commands and rejects mismatches before creating output. Cross builds need qemu-user (qemu-aarch64).
