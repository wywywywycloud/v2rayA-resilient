package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/kernel/serverObj"
	"github.com/v2rayA/v2rayA/kernel/v2ray"
)

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestHTTPLatencyClosesErrorResponseBody(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("bad gateway")}
	which := new(configure.Which)
	setHTTPLatencyResult(which, &http.Response{StatusCode: http.StatusBadGateway, Body: body}, nil, time.Now())
	if !body.closed {
		t.Fatal("response body was not closed")
	}
	if which.Latency != "BAD RESPONSE" {
		t.Fatalf("latency = %q", which.Latency)
	}
}

func TestHttpLatencyRejectsInvalidTestURLBeforeStartingCore(t *testing.T) {
	wasRunning := v2ray.ProcessManager.Running()
	_, err := TestHttpLatency(nil, time.Second, 1, false, "%")
	var coded *common.CodedError
	if !errors.As(err, &coded) || coded.Code != "INVALID_TEST_URL" {
		t.Fatalf("error = %v, want INVALID_TEST_URL", err)
	}
	if got := v2ray.ProcessManager.Running(); got != wasRunning {
		t.Fatalf("core running state changed from %v to %v", wasRunning, got)
	}
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "v2raya-service-test-*")
	if err != nil {
		panic(err)
	}
	conf.GetEnvironmentConfig().Config = dir
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestHttpLatencyDropsSubscriptions(t *testing.T) {
	input := []*configure.Which{{NodeRef: configure.NodeRef{TYPE: configure.SubscriptionType, ID: 1}}}
	got, err := TestHttpLatency(input, time.Second, 1, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("subscription remained in latency results: %+v", got)
	}
	if input[0].Latency != "" {
		t.Fatalf("subscription was probed: %q", input[0].Latency)
	}
}

func TestSupportCheckLeavesNoProducerBehind(t *testing.T) {
	obj := &serverObj.SOCKS{Server: "127.0.0.1", Port: 1080, Protocol: "socks5", Name: "probe"}
	if _, err := isSupportedObj(obj); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		if _, err := isSupportedObj(obj); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("goroutines grew from %d to %d over 20 support checks", before, after)
	}
}

func TestReplaceOutboundConnectionsWritesTheGroupOnce(t *testing.T) {
	const outbound = "replace-test"
	if err := configure.AddOutbound(outbound); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = configure.RemoveOutbound(outbound) })
	members := []configure.NodeRef{
		{TYPE: configure.SubscriptionServerType, ID: 2, Sub: 0},
		{TYPE: configure.ServerType, ID: 1, Sub: 7},
		{TYPE: configure.SubscriptionServerType, ID: 2, Sub: 0},
	}
	if err := ReplaceOutboundConnections(outbound, members); err != nil {
		t.Fatal(err)
	}
	got := configure.GetConnectedServersByOutbound(outbound)
	if got == nil || got.Len() != 2 || got.Get()[0].ID != 2 || got.Get()[1].TYPE != configure.ServerType || got.Get()[1].Sub != 0 {
		t.Fatalf("stored group %+v", got)
	}
	if err := ReplaceOutboundConnections(outbound, nil); err != nil {
		t.Fatal(err)
	}
	if got := configure.GetConnectedServersByOutbound(outbound); got != nil && got.Len() != 0 {
		t.Fatalf("group not cleared: %+v", got.Get())
	}
}

func TestDashboardProbeRejectsTCPAcceptWithoutProxyTraffic(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			time.Sleep(200 * time.Millisecond)
		}
	}()
	result := probeHTTPMode(context.Background(), listener.Addr().String(), "http://probe.invalid/", 50*time.Millisecond, false)
	if result.err == nil {
		t.Fatal("TCP accept was treated as a working proxy")
	}
	<-done
}

func TestDashboardProbeMeasuresResponseAndRejectsFailure(t *testing.T) {
	for _, status := range []int{204, 302, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(80 * time.Millisecond)
				w.WriteHeader(status)
			}))
			defer proxy.Close()
			result := probeHTTPMode(context.Background(), strings.TrimPrefix(proxy.URL, "http://"), "http://probe.invalid/", time.Second, false)
			if status != 204 {
				if result.err == nil {
					t.Fatalf("HTTP %d was treated as healthy", status)
				}
			} else if result.err != nil || result.latency < 80*time.Millisecond {
				t.Fatalf("response delay not measured: %+v", result)
			}
		})
	}
}

// Run on the disposable OpenWrt VM against the installed package core.
func TestDashboardProbeInstalledCore(t *testing.T) {
	core := os.Getenv("V2RAYA_PROBE_TEST_CORE")
	if core == "" {
		t.Skip("set V2RAYA_PROBE_TEST_CORE for the VM integration test")
	}
	old := conf.GetEnvironmentConfig().V2rayBin
	conf.GetEnvironmentConfig().V2rayBin = core
	defer func() { conf.GetEnvironmentConfig().V2rayBin = old }()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "CONNECT required", 400)
			return
		}
		upstream, err := net.Dial("tcp", strings.TrimPrefix(target.URL, "http://"))
		if err != nil {
			http.Error(w, "upstream failed", 502)
			return
		}
		downstream, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			upstream.Close()
			return
		}
		defer downstream.Close()
		defer upstream.Close()
		buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		buffered.Flush()
		go func() { io.Copy(upstream, buffered); upstream.Close() }()
		io.Copy(downstream, upstream)
	}))
	defer proxy.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no working upstream", http.StatusBadGateway)
	}))
	defer broken.Close()
	for _, tc := range []struct {
		name, address string
		healthy       bool
	}{
		{"working", proxy.URL, true}, {"accepts-tcp-but-dead", broken.URL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node, err := serverObj.NewHTTP(tc.address)
			if err != nil {
				t.Fatal(err)
			}
			legacy := &configure.Which{NodeRef: configure.NodeRef{TYPE: configure.ServerType, ID: 1}}
			loc := configure.LocatorOf([]configure.ServerRaw{{ServerObj: node}}, nil)
			if err := legacy.Ping(loc, time.Second); err != nil {
				t.Fatal(err)
			}
			t.Logf("TCP-only measurement: %s", legacy.Latency)
			result := probeSubscriptionServerMeasurementMode(context.Background(), node, target.URL, 3*time.Second, false)
			if tc.healthy {
				if result.err != nil || result.latency < 120*time.Millisecond {
					t.Fatalf("measurement: %+v", result)
				}
				t.Logf("verified HTTP response through installed core in %v", result.latency)
			} else if result.err == nil {
				t.Fatal("dead proxy incorrectly accepted")
			} else {
				t.Logf("dead proxy rejected: %v", result.err)
			}
		})
	}
}

func TestPingDropsStaleReference(t *testing.T) {
	result, err := Ping(context.Background(), []*configure.Which{{NodeRef: configure.NodeRef{TYPE: configure.SubscriptionServerType, ID: 999999, Sub: 999999}}}, time.Second)
	if err != nil || len(result) != 0 {
		t.Fatalf("stale reference was not discarded: %v %v", result, err)
	}
}

func TestPingDropsSubscriptionsWithoutStartingCore(t *testing.T) {
	result, err := Ping(context.Background(), []*configure.Which{{NodeRef: configure.NodeRef{TYPE: configure.SubscriptionType, ID: 1}}}, time.Second)
	if err != nil || len(result) != 0 {
		t.Fatalf("result=%v error=%v", result, err)
	}
}

func TestPingHonorsRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Ping(ctx, nil, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
}
