package swe

import (
	"errors"
	"net/http"
	"sync"
)

type Context struct {
	Request  *http.Request
	Response http.ResponseWriter

	values sync.Map
	chain  []HandlerFunc
	index  int
}

type HandlerFunc func(*Context)

var ctxPool sync.Pool = sync.Pool{
	New: func() any { return &Context{} },
}

func acquireContext(r *http.Request, w http.ResponseWriter, handlers ...HandlerFunc) *Context {
	ctx := ctxPool.Get().(*Context)
	ctx.reset()
	ctx.Request = r
	ctx.Response = w
	ctx.chain = handlers
	return ctx
}

func releaseContext(ctx *Context) {
	ctx.reset()
	ctxPool.Put(ctx)
}

func (ctx *Context) reset() {
	ctx.Request = nil
	ctx.Response = nil
	ctx.values = sync.Map{}
	ctx.chain = nil
	ctx.index = -1
}

func (ctx *Context) Put(key, value any) {
	ctx.values.Store(key, value)
}

func (ctx *Context) Get(key any) (any, bool) {
	return ctx.values.Load(key)
}

func CtxValue[T any](ctx *Context, key any) (ret T, ok bool) {
	if ctx == nil {
		return
	}
	value, firstOk := ctx.Get(key)
	if !firstOk {
		return
	}
	if tmpValue, tmpOk := value.(T); tmpOk {
		return tmpValue, true
	}
	return
}

func (ctx *Context) Next() {
	ctx.index++
	defer func() { ctx.index-- }()
	if len(ctx.chain) <= ctx.index {
		panic(errors.New("context no handler"))
	}
	ctx.chain[ctx.index](ctx)
}
