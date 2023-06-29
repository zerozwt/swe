package swe

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

type APIServer struct {
	handlers map[string]http.Handler
	lock     sync.RWMutex
}

func NewAPIServer() *APIServer {
	return &APIServer{
		handlers: make(map[string]http.Handler),
	}
}

func (s *APIServer) RegisterHandler(path string, handler HandlerFunc, middlewares ...HandlerFunc) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.handlers[path] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := acquireContext(r, w, append(middlewares, handler)...)
		defer releaseContext(ctx)
		ctx.Next()
	})
}

func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if handler, ok := s.handlers[r.URL.Path]; ok {
		handler.ServeHTTP(w, r)
		return
	}
	CtxLogger(nil).Error("request to API %s failed: handler not registered", r.URL.Path)
	http.NotFound(w, r)
}

// -----------------------------------------------------------------------------

type FileServer struct {
	root      string
	tryFile   string
	forbidDir bool
}

func (s *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") || strings.Contains(r.URL.Path, "/.") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.URL.Path != "/" && len(r.URL.Path) > 0 {
		target := filepath.Join(s.root, r.URL.Path[1:])
		info, err := os.Stat(target)
		if err != nil {
			if len(s.tryFile) > 0 {
				r.URL.Path = s.tryFile
			} else {
				http.NotFound(w, r)
				return
			}
		}
		if info.IsDir() && s.forbidDir {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	http.FileServer(http.Dir(s.root)).ServeHTTP(w, r)
}

func NewFileServer(root, tryFile string, forbidDir bool) *FileServer {
	return &FileServer{
		root:      root,
		tryFile:   tryFile,
		forbidDir: forbidDir,
	}
}

// -----------------------------------------------------------------------------

type Engine struct {
	apiPrefix string

	api  *APIServer
	file *FileServer

	server *http.Server
	closed int32
}

func NewEngine(apiPrefix string, apiServer *APIServer, fileServer *FileServer) *Engine {
	return &Engine{
		apiPrefix: apiPrefix,
		api:       apiServer,
		file:      fileServer,
	}
}

func (s *Engine) Serve(addr string) {
	s.server = &http.Server{Addr: addr, Handler: s}
	s.server.ListenAndServe()
}

func (s *Engine) Close() error {
	if atomic.AddInt32(&s.closed, 1) == 1 {
		return s.server.Close()
	}
	return nil
}

func (s *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, s.apiPrefix) {
		s.api.ServeHTTP(w, r)
	}
	s.file.ServeHTTP(w, r)
}
