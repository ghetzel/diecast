package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gorilla/websocket"
	"github.com/husobee/vestigo"
)

// since the Routable interface is for the benefit of external packages, lets
// make sure Server actually implements it at compile time
var _compileTimeCheckRoutable Routable = new(Server)

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
	P(req *http.Request, param string, fallback ...interface{}) typeutil.Variant
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

func (self *Server) handlersEnsureRouter() {
	if self.userRouter == nil {
		self.userRouter = vestigo.NewRouter()
	}
}

// Return the value of a URL parameter within a given request handler.
func (self *Server) P(req *http.Request, param string, fallback ...interface{}) typeutil.Variant {
	if v := vestigo.Param(req, param); v != `` {
		return typeutil.V(v)
	} else if len(fallback) > 0 {
		return typeutil.V(fallback[0])
	} else {
		return typeutil.V(nil)
	}
}

// Add a handler for an HTTP GET endpoint.
func (self *Server) Get(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodGet, route, handler)
}

func (self *Server) Websocket(route string, wsHandler WebsocketHandlerFunc, upgradeHeaders http.Header) {
	self.addHandler(http.MethodGet, route, func(w http.ResponseWriter, req *http.Request) {
		if conn, err := DefaultUpgrader.Upgrade(w, req, upgradeHeaders); err == nil {
			wsHandler(w, req, (*WebsocketConn)(conn))
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

// Add a handler for an HTTP HEAD endpoint.
func (self *Server) Head(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodHead, route, handler)
}

// Add a handler for an HTTP POST endpoint.
func (self *Server) Post(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodPost, route, handler)
}

// Add a handler for an HTTP PUT endpoint.
func (self *Server) Put(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodPut, route, handler)
}

// Add a handler for an HTTP DELETE endpoint.
func (self *Server) Delete(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodDelete, route, handler)
}

// Add a handler for an HTTP PATCH endpoint.
func (self *Server) Patch(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodPatch, route, handler)
}

// Add a handler for an HTTP OPTIONS endpoint.
func (self *Server) Options(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodOptions, route, handler)
}

// Add a handler for an HTTP CONNECT endpoint.
func (self *Server) Connect(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodConnect, route, handler)
}

// Add a handler for an HTTP TRACE endpoint.
func (self *Server) Trace(route string, handler http.HandlerFunc) {
	self.addHandler(http.MethodTrace, route, handler)
}

// Add a handler for an endpoint (any HTTP method.)
func (self *Server) HandleFunc(route string, handler http.HandlerFunc) {
	self.addHandler(``, route, handler)
}

// Add a handler function for an endpoint (any HTTP method.)
func (self *Server) Handle(route string, handler http.Handler) {
	self.hasUserRoutes = true
	self.handlersEnsureRouter()
	self.userRouter.Handle(route, handler)
}

func (self *Server) addHandler(verb string, route string, handler http.HandlerFunc) {
	self.hasUserRoutes = true
	self.handlersEnsureRouter()

	if oah := self.OnAddHandler; oah != nil {
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
		self.userRouter.Add(verb, route, func(w http.ResponseWriter, req *http.Request) {
			handler(w, req)
		})
	} else {
		self.userRouter.HandleFunc(route, handler)
	}
}
