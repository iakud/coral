package coral

import (
	"reflect"
	"regexp"
	"runtime"
	"strconv"
)

type route struct {
	pattern string
	cr      *regexp.Regexp
	method  string
	handler reflect.Value
}

type Router struct {
	routes []route
}

func NewRouter() *Router {
	return &Router{
		routes: make([]route, 0),
	}
}

func (this *Router) Get(pattern string, handler interface{}) error {
	return this.addRoute(pattern, "GET", handler)
}

func (this *Router) Post(pattern string, handler interface{}) error {
	return this.addRoute(pattern, "POST", handler)
}

func (this *Router) Put(pattern string, handler interface{}) error {
	return this.addRoute(pattern, "PUT", handler)
}

func (this *Router) Delete(pattern string, handler interface{}) error {
	return this.addRoute(pattern, "DELETE", handler)
}

func (this *Router) Match(pattern string, method string, handler interface{}) error {
	return this.addRoute(pattern, method, handler)
}

func (this *Router) addRoute(pattern string, method string, handler interface{}) error {
	cr, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	if fv, ok := handler.(reflect.Value); ok {
		this.routes = append(this.routes, route{pattern: pattern, cr: cr, method: method, handler: fv})
	} else {
		this.routes = append(this.routes, route{pattern: pattern, cr: cr, method: method, handler: reflect.ValueOf(handler)})
	}
	return nil
}

func (this *Router) routeHandler(ctx *Context, path string) {
	r := ctx.Request

	for i := 0; i < len(this.routes); i++ {
		route := this.routes[i]
		cr := route.cr
		//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
		if r.Method != route.method && !(r.Method == "HEAD" && route.method == "GET") {
			continue
		}

		if !cr.MatchString(path) {
			continue
		}
		match := cr.FindStringSubmatch(path)

		if len(match[0]) != len(path) {
			continue
		}

		var args []reflect.Value
		handlerType := route.handler.Type()
		if requiresContext(handlerType) {
			args = append(args, reflect.ValueOf(ctx))
		}
		for _, arg := range match[1:] {
			args = append(args, reflect.ValueOf(arg))
		}

		ret, err := safelyCall(ctx, route.handler, args)
		if err != nil {
			//there was an error or panic while calling the handler
			ctx.Abort(500, "Server Error")
		}
		if len(ret) == 0 {
			return
		}

		sval := ret[0]

		var content []byte

		if sval.Kind() == reflect.String {
			content = []byte(sval.String())
		} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
			content = sval.Interface().([]byte)
		}
		if _, haveType := ctx.ResponseWriter.Header()["Content-Type"]; !haveType {
			ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
		}
		ctx.SetHeader("Content-Length", strconv.Itoa(len(content)))
		_, err = ctx.ResponseWriter.Write(content)
		if err != nil {
			ctx.Server.logln("Error during write: ", err)
		}

		return
	}
	ctx.Abort(404, "Page not found")
}

var contextType reflect.Type

func init() {
	contextType = reflect.TypeOf(Context{})
}

func requiresContext(handlerType reflect.Type) bool {
	//if the method doesn't take arguments, no
	if handlerType.NumIn() == 0 {
		return false
	}
	//if the first argument is not a pointer, no
	a0 := handlerType.In(0)
	if a0.Kind() != reflect.Ptr {
		return false
	}
	//if the first argument is a context, yes
	if a0.Elem() == contextType {
		return true
	}
	return false
}

func safelyCall(ctx *Context, function reflect.Value, args []reflect.Value) (resp []reflect.Value, e interface{}) {
	defer func() {
		if err := recover(); err != nil {
			e = err
			resp = nil
			ctx.Server.logln("Handler crashed with error", err)
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				ctx.Server.logln(file, line)
			}
		}
	}()
	return function.Call(args), nil
}
