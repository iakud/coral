package coral

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"
)

type Server struct {
	Router *Router

	StaticDir string

	CookieSecret string
	Profiler     bool

	ErrorLog *log.Logger
}

func (this *Server) Run(addr string) {
	mux := http.NewServeMux()
	if this.Profiler {
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	}
	mux.Handle("/", this)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	this.logf("coral serving %s\n", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		this.fatalln("Failed to listen:", err)
	}
	this.fatalln("Failed to serve:", server.Serve(l))
}

func (this *Server) RunTLS(addr string, config *tls.Config) {
	mux := http.NewServeMux()
	mux.Handle("/", this)

	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: config,
	}
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		this.fatalln("Failed to listen:", err)
	}
	this.fatalln("Failed to serve:", server.Serve(l))
}

func (this *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := this.Router
	if router == nil {
		router = DefaultRouter
	}

	tm := time.Now().UTC()
	//ignore errors from ParseForm because it's usually harmless.
	r.ParseForm()
	defer this.logRequest(r, tm)

	ctx := &Context{w, r, this}
	//set some default headers
	ctx.SetHeader("Server", "coral")
	ctx.SetHeader("Date", webTime(tm))

	router.routeHandler(ctx, r.URL.Path)
}

func (this *Server) logRequest(r *http.Request, sTime time.Time) {
	//log the request
	var logEntry bytes.Buffer
	requestPath := r.URL.Path

	duration := time.Now().Sub(sTime)
	var client string

	// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
	// documentation at http://golang.org/pkg/net/http/#Request
	pos := strings.LastIndex(r.RemoteAddr, ":")
	if pos > 0 {
		client = r.RemoteAddr[0:pos]
	} else {
		client = r.RemoteAddr
	}

	fmt.Fprintf(&logEntry, "%s - \033[32;1m %s %s\033[0m - %v", client, r.Method, requestPath, duration)

	if len(r.Form) > 0 {
		fmt.Fprintf(&logEntry, " - \033[37;1mParams: %v\033[0m\n", r.Form)
	}

	this.logf(logEntry.String())
}

func (this *Server) logln(args ...interface{}) {
	if this.ErrorLog != nil {
		this.ErrorLog.Println(args...)
	} else {
		log.Println(args...)
	}
}

func (this *Server) logf(format string, args ...interface{}) {
	if this.ErrorLog != nil {
		this.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (this *Server) fatalln(v ...interface{}) {
	if this.ErrorLog != nil {
		this.ErrorLog.Fatalln(v...)
	} else {
		log.Fatalln(v...)
	}
}
