package coral

import (
	"crypto/tls"
)

// default router
var DefaultRouter = NewRouter()

func Get(pattern string, handler interface{}) error {
	return DefaultRouter.addRoute(pattern, "GET", handler)
}

func Post(pattern string, handler interface{}) error {
	return DefaultRouter.addRoute(pattern, "POST", handler)
}

func Put(pattern string, handler interface{}) error {
	return DefaultRouter.addRoute(pattern, "PUT", handler)
}

func Delete(pattern string, handler interface{}) error {
	return DefaultRouter.addRoute(pattern, "DELETE", handler)
}

func Match(pattern string, method string, handler interface{}) error {
	return DefaultRouter.addRoute(pattern, method, handler)
}

// server
func Run(addr string) {
	server := &Server{}
	server.Run(addr)
}

func RunTLS(addr string, config *tls.Config) {
	server := &Server{}
	server.RunTLS(addr, config)
}
