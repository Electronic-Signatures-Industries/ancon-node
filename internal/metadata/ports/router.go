package ports

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Router interface {
	POST(path string, handler http.HandlerFunc)
	GET(path string, handler http.HandlerFunc)
	Serve(port string) error
}

func NewMuxRouter() *MuxRouter {
	return &MuxRouter{
		mux: mux.NewRouter(),
	}
}

type MuxRouter struct {
	mux *mux.Router
}

func (r *MuxRouter) POST(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, handler).Methods("POST")
}

func (r *MuxRouter) GET(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, handler).Methods("GET")
}

func (r *MuxRouter) Serve(port string) error {
	return http.ListenAndServe(port, r.mux)
}
