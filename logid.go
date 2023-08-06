package swe

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var CtxKeyLogID string = "ctx_logid"
var httpHeaderLogIDKey string = "X-Logid"
var logidIncr atomic.Uint64

func init() {
	var buf [8]byte
	rand.Read(buf[:])
	logidIncr.Store(binary.BigEndian.Uint64(buf[:]))
}

func InitLogID(ctx *Context) {
	logid := ctx.Request.Header.Get(httpHeaderLogIDKey)
	if len(logid) == 0 {
		// generate logid
		logid = generateLogID()
	}
	ctx.Put(CtxKeyLogID, logid)
	ctx.Response.Header().Set(httpHeaderLogIDKey, logid)
	ctx.Next()
}

func CtxLogID(ctx *Context) string {
	logid, _ := CtxValue[string](ctx, CtxKeyLogID)
	return logid
}

func AssignLogID(ctx *Context) {
	ctx.Put(CtxKeyLogID, generateLogID())
}

func generateLogID() string {
	now := time.Now()

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], logidIncr.Add(1))

	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d", now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second()) + hex.EncodeToString(buf[:])
}
