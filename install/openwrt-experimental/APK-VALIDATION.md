# APK validation: luca-xray-key-bot

Validated on 2026-10-17 UTC using the official OpenWrt 25.12.5 ARM64 image (r33051-f5dae5ece4, Linux 6.12.94, apk-tools 3.0.5), QEMU cortex-a53, 2 CPUs and 1 GiB RAM. Package architecture was aarch64_generic. A separate /usr disk provided test space.

Application source is b51db60a6654b50372bfe6d4f3d63a204b928fc5, matching experimental IPK 2026.09.29-experimental.1. All 674 archived application files outside install/ matched before and after the build. Service, GUI, core and dependency pins were not changed. The binary version is 2026.10.17-luca.1; APK version is 2026.10.17-r1. BUILD-MANIFEST.json records hashes and toolchains.

## Passed checks

- GUI lint, typecheck, i18n check, production build and 236 tests in 47 files.
- Service go build, go vet and go test; core go build and go vet; final static Linux ARM64 builds.
- IPK payload refactor produced byte-identical IPK output with identical input binaries.
- Signed APK packages and indexes verified for aarch64_generic and aarch64_cortex-a53.
- Bootstrap repeated without repository changes; pinned public key hash and index signature verified.
- Installation with official dependencies; upgrade from a test prototype; exact service/core version pairing and binary hashes.
- UCI settings preserved on upgrade, apk fix, removal and reinstallation. Service enabled, started, restarted, recovered through procd after SIGKILL, stopped/disabled on removal and started again after reinstallation.
- Full VM reboot: persistent /usr mounted, service automatically started, API reported both final versions and both installed binary hashes matched the build.
- LuCI Software displayed all four final packages; Services contained x-Ray VPN.
- Extended local service/core network fixture: XHTTP payload integrity, node switching, direct/proxy/block routing, UDP/TCP/verified DoH DNS, concurrent DNS, TTL/prefetch/outage/reload, group methods and recovery, 16 concurrent flows, 503 propagation, cancellation with independent request recovery and core-stop origin cleanup. Fixture result: PASS.
- Separate OpenWrt 24.10.4 VM: existing opkg bootstrap ran twice, signatures verified and xray-proxy-client-experimental 2026.09.29-experimental.1 remained discoverable.

## Scope and limitations

The cortex-a53 APK contains the same tested ARMv8 binary payload as aarch64_generic; its signatures were verified, but no physical router was tested. Network tests use local fixtures and do not establish throughput, transparent interception, prompt HTTP/1 remote cancellation or all-dead fallback behavior. Existing experimental application behavior is preserved. Raw fixture artifacts remain local because they contain generated test credentials. Sanitized network results are included in BUILD-MANIFEST.json.

The reproducible package lifecycle test is tools/test-apk-openwrt.sh and requires V2RAYA_DISPOSABLE_VM=1. Local evidence is retained in the project sibling apk-lab directory and on its disposable VM disk.
