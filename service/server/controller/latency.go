package controller

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/server/service"
)

var dashboardProbes = struct {
	sync.Mutex
	next    uint64
	cancels map[uint64]context.CancelFunc
}{cancels: make(map[uint64]context.CancelFunc)}

func registerDashboardProbe(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	dashboardProbes.Lock()
	dashboardProbes.next++
	id := dashboardProbes.next
	dashboardProbes.cancels[id] = cancel
	dashboardProbes.Unlock()
	return ctx, func() {
		cancel()
		dashboardProbes.Lock()
		delete(dashboardProbes.cancels, id)
		dashboardProbes.Unlock()
	}
}

func cancelDashboardProbes() {
	dashboardProbes.Lock()
	defer dashboardProbes.Unlock()
	for _, cancel := range dashboardProbes.cancels {
		cancel()
	}
}

func GetPingLatency(ctx *gin.Context) {
	probeCtx, unregister := registerDashboardProbe(ctx.Request.Context())
	defer unregister()
	ctx.Request = ctx.Request.WithContext(probeCtx)
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var wt []*configure.Which
	err := jsoniter.Unmarshal([]byte(ctx.Query("whiches")), &wt)
	if err != nil {
		common.ResponseError(ctx, badRequest("whiches", "query parameter \"whiches\" must be a JSON array of server items"))
		return
	}
	wt, err = service.Ping(ctx.Request.Context(), wt, 8*time.Second)
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	common.ResponseSuccess(ctx, gin.H{
		"whiches": wt,
	})
}

func GetHttpLatency(ctx *gin.Context) {
	release, ok := beginMutation(ctx)
	if !ok {
		return
	}
	defer release()

	var wt []*configure.Which
	err := jsoniter.Unmarshal([]byte(ctx.Query("whiches")), &wt)
	if err != nil {
		common.ResponseError(ctx, badRequest("whiches", "query parameter \"whiches\" must be a JSON array of server items"))
		return
	}
	wt, err = service.TestHttpLatency(wt, 8*time.Second, 32, false, ctx.Query("testUrl"))
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	common.ResponseSuccess(ctx, gin.H{
		"whiches": wt,
	})
}
