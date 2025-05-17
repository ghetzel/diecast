package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gorilla/websocket"
	"github.com/husobee/vestigo"
)

type Upgrader = websocket.Upgrader

var DefaultUpgrader = Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebsocketConn = websocket.Conn
type WebsocketHandlerFunc = func(http.ResponseWriter, *http.Request, *WebsocketConn)

type Routable interface {
	P(req *http.Request, param string, fallback ...any) typeutil.Variant
	Get(route string, handler http.HandlerFunc)
	Websocket(route string, wsHandler WebsocketHandlerFunc, upgradeHeaders http.Header)
	Head(route string, handler http.HandlerFunc)
	Post(route string, handler http.HandlerFunc)
	Put(route string, handler http.HandlerFunc)
	Delete(route string, handler http.HandlerFunc)
	Patch(route string, handler http.HandlerFunc)
	Options(route string, handler http.HandlerFunc)
	Connect(route string, handler http.HandlerFunc)
	Trace(route string, handler http.HandlerFunc)
	HandleFunc(route string, handler http.HandlerFunc)
	Handle(route string, handler http.Handler)
}

type AddHandlerFunc func(verb string, route string, handler http.HandlerFunc) (string, string, http.HandlerFunc)

func (server *Server) handlersEnsureRouter() {
	if server.userRouter == nil {
		server.userRouter = vestigo.NewRouter()
	}
}

// Return the value of a URL parameter within a given request handler.
func (server *Server) P(req *http.Request, param string, fallback ...any) typeutil.Variant {
	if v := vestigo.Param(req, param); v != `` {
		return typeutil.V(v)
	} else if len(fallback) > 0 {
		return typeutil.V(fallback[0])
	} else {
		return typeutil.V(nil)
	}
}

// Add a handler for an HTTP GET endpoint.
func (server *Server) Get(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodGet, route, handler)
}

func (server *Server) Websocket(route string, wsHandler WebsocketHandlerFunc, upgradeHeaders http.Header) {
	server.addHandler(http.MethodGet, route, func(w http.ResponseWriter, req *http.Request) {
		if conn, err := DefaultUpgrader.Upgrade(w, req, upgradeHeaders); err == nil {
			wsHandler(w, req, (*WebsocketConn)(conn))
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

// Add a handler for an HTTP HEAD endpoint.
func (server *Server) Head(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodHead, route, handler)
}

// Add a handler for an HTTP POST endpoint.
func (server *Server) Post(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodPost, route, handler)
}

// Add a handler for an HTTP PUT endpoint.
func (server *Server) Put(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodPut, route, handler)
}

// Add a handler for an HTTP DELETE endpoint.
func (server *Server) Delete(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodDelete, route, handler)
}

// Add a handler for an HTTP PATCH endpoint.
func (server *Server) Patch(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodPatch, route, handler)
}

// Add a handler for an HTTP OPTIONS endpoint.
func (server *Server) Options(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodOptions, route, handler)
}

// Add a handler for an HTTP CONNECT endpoint.
func (server *Server) Connect(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodConnect, route, handler)
}

// Add a handler for an HTTP TRACE endpoint.
func (server *Server) Trace(route string, handler http.HandlerFunc) {
	server.addHandler(http.MethodTrace, route, handler)
}

// Add a handler for an endpoint (any HTTP method.)
func (server *Server) HandleFunc(route string, handler http.HandlerFunc) {
	server.addHandler(``, route, handler)
}

// Add a handler function for an endpoint (any HTTP method.)
func (server *Server) Handle(route string, handler http.Handler) {
	server.hasUserRoutes = true
	server.handlersEnsureRouter()
	server.userRouter.Handle(route, handler)
}

func (server *Server) addHandler(verb string, route string, handler http.HandlerFunc) {
	server.hasUserRoutes = true
	server.handlersEnsureRouter()

	if oah := server.OnAddHandler; oah != nil {
		v, r, h := oah(verb, route, handler)

		if v != `` {
			verb = v
		}

		if r != `` {
			route = r
		}

		if h != nil {
			handler = h
		}
	}

	if verb != `` {
		server.userRouter.Add(verb, route, func(w http.ResponseWriter, req *http.Request) {
			handler(w, req)
		})
	} else {
		server.userRouter.HandleFunc(route, handler)
	}
}
