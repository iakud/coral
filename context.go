package coral

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Server         *Server
}

func (this *Context) AddHeader(key string, val string) {
	this.ResponseWriter.Header().Add(key, val)
}

func (this *Context) SetHeader(key string, val string) {
	this.ResponseWriter.Header().Set(key, val)
}

func (this *Context) WriteString(content string) {
	this.ResponseWriter.Write([]byte(content))
}

func (this *Context) ServeFile(name string) {
	http.ServeFile(this.ResponseWriter, this.Request, name)
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (this *Context) Abort(status int, body string) {
	this.ResponseWriter.WriteHeader(status)
	this.ResponseWriter.Write([]byte(body))
}

// Redirect is a helper method for 3xx redirects.
func (this *Context) Redirect(status int, url string) {
	this.ResponseWriter.Header().Set("Location", url)
	this.ResponseWriter.WriteHeader(status)
	this.ResponseWriter.Write([]byte("Redirecting to: " + url))
}

// Notmodified writes a 304 HTTP response
func (this *Context) NotModified() {
	this.ResponseWriter.WriteHeader(304)
}

// NotFound writes a 404 HTTP response
func (this *Context) NotFound(message string) {
	this.ResponseWriter.WriteHeader(404)
	this.ResponseWriter.Write([]byte(message))
}

//Unauthorized writes a 401 HTTP response
func (this *Context) Unauthorized() {
	this.ResponseWriter.WriteHeader(401)
}

//Forbidden writes a 403 HTTP response
func (this *Context) Forbidden() {
	this.ResponseWriter.WriteHeader(403)
}

// ContentType sets the Content-Type header for an HTTP response.
// For example, ctx.ContentType("json") sets the content-type to "application/json"
// If the supplied value contains a slash (/) it is set as the Content-Type
// verbatim. The return value is the content type as it was
// set, or an empty string if none was found.
func (this *Context) ContentType(val string) string {
	var ctype string
	if strings.ContainsRune(val, '/') {
		ctype = val
	} else {
		if !strings.HasPrefix(val, ".") {
			val = "." + val
		}
		ctype = mime.TypeByExtension(val)
	}
	if ctype != "" {
		this.ResponseWriter.Header().Set("Content-Type", ctype)
	}
	return ctype
}

func (this *Context) SetCookie(cookie *http.Cookie) {
	this.AddHeader("Set-Cookie", cookie.String())
}

func getCookieSig(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

func (this *Context) SetSecureCookie(name string, val string, age int64) {
	//base64 encode the val
	if len(this.Server.CookieSecret) == 0 {
		this.Server.logln("Secret Key for secure cookies has not been set. Please assign a cookie secret to web.Config.CookieSecret.")
		return
	}
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	vs := buf.String()
	vb := buf.Bytes()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := getCookieSig(this.Server.CookieSecret, vb, timestamp)
	cookie := strings.Join([]string{vs, timestamp, sig}, "|")
	this.SetCookie(NewCookie(name, cookie, age))
}

func (this *Context) GetSecureCookie(name string) (string, bool) {
	for _, cookie := range this.Request.Cookies() {
		if cookie.Name != name {
			continue
		}

		parts := strings.SplitN(cookie.Value, "|", 3)

		val := parts[0]
		timestamp := parts[1]
		sig := parts[2]

		if getCookieSig(this.Server.CookieSecret, []byte(val), timestamp) != sig {
			return "", false
		}

		ts, _ := strconv.ParseInt(timestamp, 0, 64)

		if time.Now().Unix()-31*86400 > ts {
			return "", false
		}

		buf := bytes.NewBufferString(val)
		encoder := base64.NewDecoder(base64.StdEncoding, buf)

		res, _ := ioutil.ReadAll(encoder)
		return string(res), true
	}
	return "", false
}
