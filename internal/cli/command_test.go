package cli

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// needed to retrieve requests that arrived at httpServer for further investigation
var requestChan = make(chan *RequestData, 1)

// the request datastructure that can be retrieved for test assertions
type RequestData struct {
	request *http.Request
	body    string
}

// set the response body the httpServer should return for the next request
var responseBody = ""

var httpServer *httptest.Server

// start the testhttp server and stop it when tests are finished
func TestMain(m *testing.M) {
	httpServer = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		// put request and body to channel for the client to investigate them
		requestChan <- &RequestData{r, string(data)}
	}))
	defer httpServer.Close()

	os.Exit(m.Run())
}
