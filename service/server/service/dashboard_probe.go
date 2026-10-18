package service

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/kernel/coreObj"
	"github.com/v2rayA/v2rayA/kernel/serverObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray"
	"github.com/v2rayA/v2rayA/kernel/v2ray/asset"
	"github.com/v2rayA/v2rayA/kernel/v2ray/where"
)

func probeDashboardBatch(parent context.Context, servers []serverObj.ServerObj, probeURL string, timeout time.Duration) []subscriptionProbeResult {
	results := make([]subscriptionProbeResult, len(servers))
	fail := func(err error) []subscriptionProbeResult {
		for i := range results {
			results[i].err = err
		}
		return results
	}
	if len(servers) == 0 {
		return results
	}
	if err := acquireProbeCore(parent); err != nil {
		return fail(err)
	}
	defer releaseProbeCore()

	if err := parent.Err(); err != nil {
		return fail(err)
	}
	u, err := url.Parse(probeURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fail(fmt.Errorf("invalid HTTP probe URL"))
	}
	bin, err := where.GetV2rayBinPath()
	if err != nil {
		return fail(err)
	}
	tmpl := v2ray.NewEmptyTemplate(configure.GetSettingNotNil())
	defer tmpl.Close()
	listeners := make([]net.Listener, 0, len(servers))
	addresses := make([]string, len(servers))
	defer func() {
		for _, listener := range listeners {
			listener.Close()
		}
	}()
	for i, server := range servers {
		if server == nil {
			results[i].err = fmt.Errorf("node cannot be probed in isolation")
			continue
		}
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fail(err)
		}
		listeners = append(listeners, listener)
		port := listener.Addr().(*net.TCPAddr).Port
		if err := tmpl.InsertMappingOutbound(server, strconv.Itoa(port), false, 0, "http"); err != nil {
			return fail(err)
		}
		tmpl.Inbounds[len(tmpl.Inbounds)-1].Listen = "127.0.0.1"
		addresses[i] = listener.Addr().String()
	}
	if len(listeners) == 0 {
		return results
	}
	// Candidate connections must not be intercepted by the main group's
	// transparent proxy, especially while that group is empty and blocking.
	tmpl.SetOutboundSockopt()
	tmpl.Routing.DomainStrategy = "AsIs"
	tmpl.Log = &coreObj.Log{Access: "none", Error: "none", Loglevel: "none"}
	file, err := os.CreateTemp("", "v2raya-subscription-*.json")
	if err != nil {
		return fail(err)
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(tmpl.ToConfigBytes()); err != nil {
		file.Close()
		return fail(err)
	}
	if err = file.Close(); err != nil {
		return fail(err)
	}
	startupTimeout := time.Duration(conf.GetEnvironmentConfig().CoreStartupTimeout) * time.Second
	if startupTimeout <= 0 {
		startupTimeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, startupTimeout+time.Duration((len(servers)+3)/4)*timeout+time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "run", "--config="+file.Name())
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+asset.GetV2rayLocationAssetOverride(), "V2RAY_CONF_GEOLOADER=memconservative")
	// Orphaned probe cores survived OOM kills and piled up (~17 MiB for
	// 3 in the field log). Pdeathsig kills them with v2raya; group-cancel
	// reaps grandchildren on normal timeouts.
	v2ray.ConfigureChildDeathSig(cmd)
	for _, listener := range listeners {
		listener.Close()
	}
	if err = cmd.Start(); err != nil {
		return fail(err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()
	readyCtx, readyCancel := context.WithTimeout(ctx, startupTimeout)
	defer readyCancel()
	address := listeners[0].Addr().String()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return fail(fmt.Errorf("probe core exited before becoming ready"))
		case <-readyCtx.Done():
			return fail(fmt.Errorf("probe core startup timeout"))
		case <-ticker.C:
			conn, e := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if e == nil {
				conn.Close()
				var wg sync.WaitGroup
				slots := make(chan struct{}, 4)
				for i, address := range addresses {
					if address == "" {
						continue
					}
					select {
					case slots <- struct{}{}:
					case <-ctx.Done():
						wg.Wait()
						return fail(ctx.Err())
					}
					wg.Add(1)
					go func(i int, address string) {
						defer wg.Done()
						defer func() { <-slots }()
						results[i] = probeHTTPMode(ctx, address, probeURL, timeout, false)
					}(i, address)
				}
				wg.Wait()
				return results
			}
		}
	}
}
