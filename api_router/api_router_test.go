package api_router

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var router *Router
var baseURL string
var gpRoute *Route
var gpRouteFn = func(ctx context.Context) {}

func TestMain(m *testing.M) {
	flag.Parse()
	startServer()
	os.Exit(m.Run())
}

func startServer() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Listening on %s\n", listener.Addr().String())

	baseURL = "http://" + listener.Addr().String()

	router = NewMuxRouter()

	gpRoute = router.GET("/GeneralPurpose", func(ctx context.Context) {
		gpRouteFn(ctx)
	})

	go func() {
		err := http.Serve(listener, router)
		if err != nil {
			panic(err.Error())
		}
	}()

	// Wait for server to start
	for {
		resp, err := http.Get(baseURL)
		if err == nil && resp.StatusCode == 404 {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}

	fmt.Printf("Server available at %s\n", baseURL)
}

func GetBody(resp *http.Response) string {
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}
	s := string(bytes)
	return strings.TrimRight(s, "\n")
}

func GET(path string) *http.Response {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		panic(err.Error())
	}
	return resp
}

func POST(path string, ctype string, text string) *http.Response {
	buf := bytes.NewBufferString(text)

	resp, err := http.Post(
		baseURL+path,
		ctype,
		buf,
	)
	if err != nil {
		panic(err.Error())
	}
	return resp
}

func TestGETWithNoResponse(t *testing.T) {
	router.GET("/EmptyResponse", func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		rctx.SetStatus(345)
	})

	resp := GET("/EmptyResponse")

	if resp.StatusCode != 345 {
		t.Errorf("Got unexpected status for GET: %d\n", resp.StatusCode)
	}
}

func TestGETWithWriteResponse(t *testing.T) {
	response_string := "response data"

	var writer ResponseTracker

	router.GET("/WriteResponse", func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		rctx.WriteResponse([]byte(response_string))
		writer = rctx.ResponseTracker()
	})

	resp := GET("/WriteResponse")
	data := GetBody(resp)

	if resp.StatusCode != 200 {
		t.Errorf("Got unexpected status for GET: %d\n", resp.StatusCode)
	}

	if data != response_string {
		t.Errorf("Got unexpected response for GET: %+v\n", data)
	}

	if writer.Status() != 200 {
		t.Errorf("writer Status != 200: %d\n", writer.Status())
	}

	if writer.Size() != int64(len(response_string)) {
		t.Errorf("writer Size != %d: %d\n",
			len(response_string), writer.Size(),
		)
	}
}

func TestGETWithWriteResponseString(t *testing.T) {
	response_string := "response string"

	var writer ResponseTracker

	router.GET("/WriteResponseString", func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		rctx.WriteResponseString(response_string)
		writer = rctx.ResponseTracker()
	})

	resp := GET("/WriteResponseString")
	data := GetBody(resp)

	if resp.StatusCode != 200 {
		t.Errorf("Got unexpected status for GET: %d\n", resp.StatusCode)
	}

	if data != response_string {
		t.Errorf("Got unexpected response for GET: %+v\n", data)
	}

	if writer.Status() != 200 {
		t.Errorf("writer Status != 200: %d\n", writer.Status())
	}

	if writer.Size() != int64(len(response_string)) {
		t.Errorf("writer Size != %d: %d\n",
			len(response_string), writer.Size(),
		)
	}
}

func TestGetCurrentRoute(t *testing.T) {
	var cur_route *Route

	gpRouteFn = func(ctx context.Context) {
		cur_route = router.RequestContext(ctx).CurrentRoute()
	}

	GET("/GeneralPurpose")

	if cur_route != gpRoute {
		t.Errorf("Got unexpected current route: %+v\n", cur_route)
	}
}

func TestGetRequest(t *testing.T) {
	var request *http.Request

	gpRouteFn = func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		request = rctx.HTTPRequest()
	}

	GET("/GeneralPurpose")

	if request.Method != "GET" {
		t.Errorf("http request method not correct: %+v\n", request)
	}

	if request.URL.String() != "/GeneralPurpose" {
		t.Errorf("http request URL not correct: %+v\n", request)
	}
}

func TestPOSTText(t *testing.T) {
	var ctype string
	post_data := "This is the POST data"

	router.POST("/POSTText", func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		ctype = rctx.Header("Content-Type")
		rctx.SetResponseHeader("Content-Type", "text/plain")
		rctx.WriteResponseString("Response:")
		io.Copy(rctx.ResponseWriter(), rctx.Body())
	})

	resp := POST("/POSTText", "text/plain", post_data)
	data := GetBody(resp)

	if resp.StatusCode != 201 {
		t.Errorf("Got unexpected status for POST: %d\n", resp.StatusCode)
	}

	if data != "Response:"+post_data {
		t.Errorf("Got unexpected response for GET: %+v\n", data)
	}

	if ctype != "text/plain" {
		t.Errorf("Content-Type is not text/plain: %s", ctype)
	}

	if hdr := resp.Header.Get("Content-Type"); hdr != "text/plain" {
		t.Errorf("Got unexpected Content-Type: %s\n", hdr)
	}
}

func TestPOSTJSON(t *testing.T) {
	var ctype string
	post_data := `{ "moo": "cow" }`

	router.POST("/POSTJSON", func(ctx context.Context) {
		rctx := router.RequestContext(ctx)
		ctype = rctx.Header("Content-Type")
		rctx.SetResponseHeader("Content-Type", "application/json")
		io.Copy(rctx.ResponseWriter(), rctx.Body())
	})

	resp := POST("/POSTJSON", "application/json", post_data)
	data := GetBody(resp)

	if resp.StatusCode != 201 {
		t.Errorf("Got unexpected status for POST: %d\n", resp.StatusCode)
	}

	if data != post_data {
		t.Errorf("Got unexpected response for GET: %+v\n", data)
	}

	if ctype != "application/json" {
		t.Errorf("Content-Type is not text/plain: %s", ctype)
	}

	if hdr := resp.Header.Get("Content-Type"); hdr != "application/json" {
		t.Errorf("Got unexpected Content-Type: %s\n", hdr)
	}
}
