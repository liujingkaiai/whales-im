package core

import (
	"errors"
	"fmt"
	"im/iface"
	"sync"

	"github.com/klintcheng/kim/wire/pkt"
)

var ErrSessionLost = errors.New("err:session lost")

type Router struct {
	handlers *FuncTree
	pool     sync.Pool
}

func NewRouter() *Router {
	r := &Router{
		handlers: NewTree(),
	}
	r.pool.New = func() interface{} {
		return BuildContext()
	}
	return r
}

func (r *Router) Handle(command string, handlers ...HandleFunc) {
	r.handlers.Add(command, handlers...)
}

func (r *Router) Serve(packet *pkt.LogicPkt, dispatcher iface.Dispatcher, cache iface.ISessionStorage, session iface.ISession) error {
	if dispatcher == nil {
		return fmt.Errorf("dispacher is nil")
	}
	if cache == nil {
		return fmt.Errorf("cache is nil")
	}
	ctx := r.pool.Get().(*Context)
	ctx.reset()
	ctx.request = packet
	ctx.Dispatcher = dispatcher
	ctx.ISessionStorage = cache
	ctx.session = session

	r.serveContext(ctx)
	r.pool.Put(ctx)
	return nil
}

func (s *Router) serveContext(ctx *Context) {
	chain, ok := s.handlers.GetPath(ctx.request.Command)
	if !ok {
		ctx.handlers = []HandleFunc{}
		ctx.Next()
		return
	}
	ctx.handlers = chain
	ctx.Next()
}

func handleNotFound(ctx Context) {
	ctx.Resp(pkt.Status_NotImplemented, &pkt.ErrorResp{Message: "NotImplemented"})
}

type FuncTree struct {
	nodes map[string]HandleFuncChain
}

func NewTree() *FuncTree {
	return &FuncTree{
		nodes: make(map[string]HandleFuncChain, 10),
	}
}

func (t *FuncTree) Add(path string, handlers ...HandleFunc) {
	t.nodes[path] = append(t.nodes[path], handlers...)
}

func (t *FuncTree) GetPath(path string) (HandleFuncChain, bool) {
	chians, ok := t.nodes[path]
	return chians, ok
}
