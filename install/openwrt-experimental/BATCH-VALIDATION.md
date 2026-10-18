# APK r5: dashboard batch and core controls

Published 2026.10.18-r5, source bd5d54dd, both binary versions 2026.10.18-luca.5. A dashboard list uses one isolated core with four concurrent HTTP requests. Starting or stopping the main core cancels queued/running dashboard probes before acquiring the mutation lock. The strict compatibility check and packaging gate from r4 remain enabled.

OpenWrt 25.12.5 ARM64 VM, same 24-node subscription: r3 list 116.890 s; final r5 list 31.188 s (earlier r5 sample 30.250 s). Unreachable nodes remain errors/timeouts. Controlled installed-core mixed batch rejects the dead proxy and independently measures healthy candidates.

During an active list check, stop completed in 0.781 s and canceled the probe; restart took 5.407 s. Start during a separate active probe also canceled it and completed in 6.406 s. No orphaned probe cores remained. Domain HTTPS returned expected content after the controls test. Full service tests/vet passed before the final start-button hook; final controller/service tests/vet passed after it. GUI and core protocol sources are unchanged.

Public installation through the original bootstrap URL passed with normal signature validation, matching installed hashes, preserved UCI/account/subscription, and valid API version status. All ten public APK/index downloads match the tested hashes. See BATCH-VALIDATION.json and BUILD-MANIFEST.json.

Timings describe this VM/network path and are not a physical-router benchmark. The real router was inspected read-only. Its overlapping WAN/LAN subnet was changed by the user; DNS and HTTPS passed afterward. No router software was changed by this task.
