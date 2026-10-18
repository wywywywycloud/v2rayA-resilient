#!/usr/bin/env python3
"""Build signed OpenWrt APK v3 packages from the experimental payloads."""
import argparse
import hashlib
import importlib.util
import os
from pathlib import Path
import re
import shlex
import subprocess
import tempfile

spec = importlib.util.spec_from_file_location("ipk_payloads", Path(__file__).with_name("package-current.py"))
payload_module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(payload_module)

def run(*args):
    subprocess.run(list(map(str, args)), check=True)

def verify_binary_versions(service, core, runner=""):
    prefix = shlex.split(runner)
    service_version = subprocess.check_output([*prefix, str(service.resolve()), "--version"], text=True, timeout=30).strip()
    core_output = subprocess.check_output([*prefix, str(core.resolve()), "version"], text=True, timeout=30)
    match = re.search(r"^V2RAYA_CORE\s+(\S+)", core_output, re.MULTILINE)
    core_version = match.group(1) if match else ""
    if not service_version or service_version != core_version:
        raise ValueError(f"Refusing APK with mismatched binary versions: service={service_version}, core={core_version}")
    if service_version.startswith(("debug", "unstable")):
        raise ValueError("Release APK must not use a development version")
    return service_version


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--service", type=Path, required=True)
    p.add_argument("--core", type=Path, required=True)
    p.add_argument("--version", required=True)
    p.add_argument("--binary-runner", default="", help="Runner for cross-built binaries, e.g. qemu-aarch64")
    p.add_argument("--arch", choices=["aarch64_generic", "aarch64_cortex-a53"], required=True)
    p.add_argument("--apk", type=Path, required=True)
    p.add_argument("--key", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    a = p.parse_args()
    if os.geteuid() != 0:
        p.error("Run under root or fakeroot so package files are owned by root")
    if not re.fullmatch(r"[0-9]+(?:\.[0-9]+)*-r[0-9]+", a.version):
        p.error("Use an APK version such as 2026.10.18-r1")
    verify_binary_versions(a.service, a.core, a.binary_runner)
    a.apk, a.key = a.apk.resolve(), a.key.resolve()
    a.output.mkdir(parents=True, exist_ok=False)
    a.output = a.output.resolve()
    core, service, luci = payload_module.payloads(a.service, a.core)
    root = Path(__file__).resolve().parents[1]
    core["usr/share/licenses/xray-proxy-client-experimental-core/LICENSE"] = ((root.parents[1] / "core/LICENSE").read_bytes(), 0o644)
    service["usr/share/licenses/xray-proxy-client-experimental-service/LICENSE"] = ((root.parents[1] / "LICENSE").read_bytes(), 0o644)
    stop = '#!/bin/sh\n[ -n "$IPKG_INSTROOT" ] && exit 0\n[ ! -x /etc/init.d/v2raya ] || /etc/init.d/v2raya stop\nexit 0\n'
    remove = stop.rsplit("exit 0\n", 1)[0] + "/etc/init.d/v2raya disable\nexit 0\n"
    start = '#!/bin/sh\n[ -n "$IPKG_INSTROOT" ] && exit 0\n/etc/init.d/v2raya enable\ncase "$(uci -q get v2raya.config.enabled)" in 1|on|true|yes|enabled) if /etc/init.d/v2raya running; then /etc/init.d/v2raya restart; else /etc/init.d/v2raya start; fi ;; esac\nexit 0\n'
    clear = '#!/bin/sh\n[ -n "$IPKG_INSTROOT" ] && exit 0\nrm -f /tmp/luci-indexcache\nexit 0\n'
    rows = [
        ("xray-proxy-client-experimental-core", core, "libc !luca-xray-key-bot-core !v2raya-core !v2raya-resilient-core !v2raya-levin-core", "MPL-2.0", "Matching Xray core for xray-proxy-client-experimental", {}),
        ("xray-proxy-client-experimental-service", service, f"libc !luca-xray-key-bot-service ca-bundle kmod-tun kmod-nft-tproxy v2ray-geoip v2ray-geosite xray-proxy-client-experimental-core={a.version} !v2raya !v2raya-resilient !v2raya-levin", "AGPL-3.0-only", "Experimental x-Ray VPN service", {"pre-upgrade": stop, "pre-deinstall": remove, "post-install": start, "post-upgrade": start}),
        ("luci-app-xray-proxy-client-experimental", luci, f"!luci-app-luca-xray-key-bot luci-light xray-proxy-client-experimental-service={a.version} !luci-app-v2raya !luci-app-v2raya-resilient !luci-app-v2raya-levin", "Apache-2.0", "LuCI controls for xray-proxy-client-experimental", {"post-install": clear, "post-upgrade": clear}),
        ("xray-proxy-client-experimental", {}, f"luci-app-xray-proxy-client-experimental={a.version}", "AGPL-3.0-only", "Complete x-Ray VPN experimental client for OpenWrt 25.12", {}),
    ]
    packages = []
    with tempfile.TemporaryDirectory(prefix="luca-apk-") as tmp:
        tmp = Path(tmp)
        for name, files, depends, license_name, description, scripts in rows:
            stage = tmp / name / "files"
            stage.mkdir(parents=True)
            files = dict(files)
            meta = "lib/apk/packages/" + name
            if name.endswith("-service"):
                config = "/etc/config/v2raya"
                files[meta + ".conffiles"] = ((config + "\n").encode(), 0o644)
                files[meta + ".conffiles_static"] = ((config + " " + hashlib.sha256(files[config[1:]][0]).hexdigest() + "\n").encode(), 0o644)
            files[meta + ".list"] = (("".join("/" + f + "\n" for f in sorted(files))).encode(), 0o644)
            for filename, (data, mode) in files.items():
                path = stage / filename
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
                path.chmod(mode)
            output = a.output / f"{name}-{a.version}.apk"
            cmd = [a.apk, "mkpkg", "--info", f"name:{name}", "--info", f"version:{a.version}", "--info", f"arch:{a.arch}", "--info", f"description:{description}", "--info", f"license:{license_name}", "--info", "url:https://github.com/wywywywycloud/v2rayA-current/tree/main-apk-feed", "--info", f"depends:{depends}", "--files", stage, "--sign-key", a.key, "--output", output]
            for event, text in scripts.items():
                script = tmp / name / event
                script.write_text(text)
                cmd += ["--script", f"{event}:{script}"]
            run(*cmd)
            packages.append(output)
        run(a.apk, "--keys-dir", a.key.parent, "mkndx", "--sign-key", a.key, "--output", a.output / "packages.adb", *packages)
    (a.output / "SHA256SUMS").write_text("".join(f"{hashlib.sha256(f.read_bytes()).hexdigest()}  {f.name}\n" for f in [*packages, a.output / "packages.adb"]))
if __name__ == "__main__":
    main()
