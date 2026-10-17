# OpenWrt 25.12 APK update: 2026.10.18-r3

Signed ARM64 packages with verified HTTPS node probes and kmod-tun dependency. See [installation and migration](openwrt-25.12/resilient/README.md). The bootstrap URL remains unchanged; OpenWrt 24.10 opkg packages below are unaffected.

# x-Ray VPN experimental

Install **xray-proxy-client-experimental** from this signed feed. The metapackage pulls the exact matching service, core and LuCI version `2026.09.29-experimental.1`, plus their declared dependencies from your official OpenWrt repositories.

If this Resilient feed is already configured, click **System > Software > Update lists**, then search for **xray-proxy-client-experimental** and install it. From SSH:

```sh
opkg update
opkg install xray-proxy-client-experimental
```

For a new feed setup, run the existing `add-resilient-feed.sh`; the feed URL remains unchanged. The signing key was rotated on 2026-09-29 (old `9478c50315c92021` lost, new `1c624cadb7ee79e7`): existing installations must re-run the script once before `opkg update`. Keep your device's official repositories enabled. Kernel modules must come from the repository matching your installed kernel; never force dependency or architecture checks.

This replaces the existing Resilient application using its compatible package names and configuration paths. It is not a parallel installation. Since these same package names now point to the newer experimental version, they also appear as updates for installed Resilient packages. Back up your configuration before updating.

The release was tested on OpenWrt 24.10.4 ARM64 (`aarch64_cortex-a53`). No new nine-version or physical-router test is claimed. Existing package files remain available for rollback. The complete dependency chain and validation are in [the experimental release](https://github.com/wywywywycloud/v2raya-openwrt-current/releases/tag/experimental-2026.09.29.1).

---
# v2rayA Resilient

For OpenWrt 24.10.0–24.10.8 on `aarch64_cortex-a53`, use the [Resilient signed feed and GUI installation guide](openwrt-24.10/resilient/README.md).
Install `luci-app-v2raya-resilient` in LuCI Software.

---

# Signed v2rayA fork packages for OpenWrt

Binary opkg feed for OpenWrt 24.10.4 and `aarch64_cortex-a53` (Cudy TR3000). The `v2raya-fork` bundle installs the tested v2rayA fork with its embedded GUI, Xray 25.1.30-r1, geo assets, LuCI and `luci-app-v2raya`. Dependencies come from the router's official OpenWrt feeds.

Feed URL:

```text
https://raw.githubusercontent.com/wywywywycloud/v2rayA/openwrt-feed/openwrt-24.10/aarch64_cortex-a53
```

Public signing key fingerprint: `9f02e659f24749fa`.
Public key SHA-256: `6ef5500355caf6da06e020818151ee1a6533336387450b7ac28dcaba5540d979`.

The installer verifies the key checksum and package index signature, retains existing official feeds, installs the complete package set, and enables the v2rayA service. It does not disable signature verification or force architecture/space checks. Existing v2rayA settings are retained and backed up in `/tmp` before installation.

See [installation and maintenance](https://github.com/wywywywycloud/v2rayA/blob/fix/openwrt-subscription-failover/install/openwrt/feed/README.md) for the verified installation commands and rollback instructions.

Application source: [fork branch](https://github.com/wywywywycloud/v2rayA/tree/fix/openwrt-subscription-failover).
Application commit: `c39977ab7061c6b87b6b8c025f733d141fb3c47e`.
Application package SHA-256: `bbc005d31b5837a601356ed231881ac491f8ef055a2b7bf123d21c31665d4031`.

The application is AGPL-3.0-only. Upstream dependencies retain their own licenses. The signing private key is never included in this branch.

Validated bundle: `2.2.7.3-r4.failover3.feed1`. See [validation results](https://github.com/wywywywycloud/v2rayA/blob/fix/openwrt-subscription-failover/install/openwrt/feed/VALIDATION.md).

## Current-source release: 2.5.7-recovery.2

The original 2.2.x feed above remains available. The separate current-source
feed is `openwrt-24.10/current/aarch64_cortex-a53`. It contains v2rayA, its
matching `v2raya-core` and current LuCI sources, tested on OpenWrt 24.10.4.
Use `install-current-feed.sh`; it preserves settings, verifies the signing key
and index, replaces the old feed entry and removes only the old version-pinning
`v2raya-fork` metapackage when present. It installs `luci-app-v2raya`, which pulls
in the matching application, core and official dependencies.

Service/core source: https://github.com/wywywywycloud/v2rayA/tree/b3c6789330daf3c25f4aaa5464ad2385a7e3c35a

Package pipeline/LuCI source: https://github.com/wywywywycloud/v2raya-openwrt-current/tree/dca1f5c2172ec6c303418c68cc95e020aff4afcc

Validation uses a generic ARM64 virtual machine, not physical Cudy hardware.
The legacy SDK recipe remains available; the current packages use the documented
Go 1.26 / Node 24 source-build pipeline. Do not select standalone `xray-core`
as the executable for this release: current v2rayA requires `v2raya_core`.
