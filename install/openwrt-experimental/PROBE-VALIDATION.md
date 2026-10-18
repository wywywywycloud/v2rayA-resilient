# Proxy latency regression — 2026-10-18

The dashboard used a TCP connect timer as its node latency. A listening endpoint can accept TCP without completing the proxy handshake or forwarding any traffic. QEMU user networking and an upstream proxy can make this especially misleading.

`pingLatency` now measures an HTTPS request through an isolated core configured for the selected node. Only a successful 2xx response is a measurement. Failures become an error, not a low latency. The existing serialized probe-core owner limits memory use; the main core is not restarted by this endpoint. The request context cancels queued/running probes when the caller disconnects. Results are saved so a page reload retains the verified measurement. HTTP timeout is eight seconds per candidate, excluding core startup; checking many nodes therefore takes longer than TCP-only ping.

The APK also requires `kmod-tun`, following the earlier VM reproduction of a missing `/dev/net/tun` startup failure. The core source and core binary are unchanged.

## Reproduction and results

- OpenWrt 25.12.5 ARM64, QEMU TCG with user networking; signed local APK candidates, subsequently published unchanged as 2026.10.18-r3.
- Controlled dead proxy: TCP-only measurement 2 ms; request through the installed core returned HTTP 503 and was rejected.
- Controlled working proxy: a deliberately delayed HTTP response was measured at approximately 269 ms rather than the 4 ms TCP connection.
- Original and fixed local packages passed the extended service/core network harness: payload transfer, DNS UDP/TCP/verified DoH, cache expiry/outage/recovery, routing, group selection, 16 concurrent flows, HTTP 503 and cancellation cases. This is functional validation, not a performance or transparent interception benchmark.
- User-authorized private subscription: original TCP checks reported 4–7 ms for all 24 nodes. Verified HTTPS checks passed four nodes at 493–1010 ms on this VM path; twenty failed the proxy exchange. All four successful nodes subsequently returned the expected HTTPS page. Bounded downloads of 64 KiB and 256 KiB also succeeded.
- One failed node, same configuration and assets: both the exact core from the working IPK release and the APK core returned HTTP 503 with REALITY certificate rejection. This result does not establish that the node is unavailable on every network path.
- Automatic selection with one failed and one healthy real node selected the healthy node and transferred HTTPS data. After leaving only the failed member, no eligible/active node remained and traffic was blocked. Restoring the healthy member restored HTTPS after the asynchronous selection pass.
- The initial Python traffic fixture received HTTP 403 from the speed endpoint with its default User-Agent. Repeating with the application's Go User-Agent downloaded all 262144 bytes. No application change was made for this fixture-specific failure.

The sanitized measurements are in [PROBE-VALIDATION.json](PROBE-VALIDATION.json). Subscription URLs, credentials, generated private configurations and raw real-node logs are excluded from Git.

Final r3 validation also passed the full local network harness. An actual API request aborted after 1.016 seconds left zero probe cores; the next real-node check completed in 1.141 seconds and reported 709 ms. Signed APKs were built and verified for aarch64_generic and aarch64_cortex-a53.

## Checks

Service: `go build ./...`, `go vet ./...`, `go test ./...`.
Core: `go build ./...`, `go vet ./...`.
GUI: lint, typecheck, i18n-check, 236 tests in 47 files, production build. Lint had no errors; the Windows checkout's existing CRLF formatting produced warnings.

The committed core integration test is opt-in. From `service`, cross-compile it with:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go test -c -o probe-service.test ./server/service
```

On a disposable OpenWrt VM with the matching package installed:

```sh
V2RAYA_PROBE_TEST_CORE=/usr/bin/v2raya_core \
XRAY_LOCATION_ASSET=/usr/share/v2ray \
./probe-service.test -test.run 'TestDashboardProbe|TestPing' -test.v -test.timeout 90s
```

These HTTPS timings include the proxy/TLS exchange, destination response and emulation overhead. They are not ICMP RTT and should not be presented as the physical router's exact latency. A failed test means that this node could not reach the test URL from the tested path at that time. External test-target availability still matters.

BUILD-MANIFEST.json now describes the published r3 artifacts. Public installation through the original bootstrap URL passed on the same VM; configuration and database hashes were preserved. Both architecture package/index downloads match the tested SHA256 values.
