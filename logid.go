package swe

import (
	"fmt"
	"sync/atomic"
	"time"
)

var CtxKeyLogID string = "ctx_logid"
var httpHeaderLogIDKey string = "X-Logid"
var logidIncr atomic.Int64

func InitLogID(ctx *Context) {
	logid := ctx.Request.Header.Get(httpHeaderLogIDKey)
	if len(logid) == 0 {
		// generate logid
		ts := time.Now().Unix()
		low := logidIncr.Add(1)
		logid = fmt.Sprint(ts<<20 | (low & 0xFFFFF))
	}
	ctx.Put(CtxKeyLogID, logid)
	ctx.Response.Header().Set(httpHeaderLogIDKey, logid)
	ctx.Next()
}

func CtxLogID(ctx *Context) string {
	logid, _ := CtxValue[string](ctx, CtxKeyLogID)
	return logid
}
