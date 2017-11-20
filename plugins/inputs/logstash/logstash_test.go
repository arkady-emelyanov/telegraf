package logstash

import (
	"fmt"
	"os"
	"io/ioutil"
	"net/http/httptest"
	"net/http"
	"testing"

	"github.com/influxdata/telegraf/testutil"
)

// respond with version number
func getResponseJSON(file string) ([]byte, int) {
	jsonFile := fmt.Sprintf("./testdata/%s.json", file)

	code := 200
	_, err := os.Stat(jsonFile)
	if err != nil {
		code = 404
		jsonFile = "./testdata/error.json"
	}

	// respond with file
	b, _ := ioutil.ReadFile(jsonFile)
	return b, code
}

// return mocked HTTP server
func getHTTPServerVersion(version string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, code := getResponseJSON(version)
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

// return mocked HTTP server with basic auth
func getHTTPServerBasicAuthVersion(version string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		username, password, authOK := r.BasicAuth()
		if authOK == false {
			http.Error(w, "Not authorized", 401)
			return
		}

		if username != "test" && password != "test" {
			http.Error(w, "Not authorized", 401)
			return
		}

		// ok, continue
		body, code := getResponseJSON(version)
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

//
func TestLogstash5(t *testing.T) {
	s := getHTTPServerVersion("logstash_5")
	defer s.Close()

	plugin := &logstash{
		Servers: []string{s.URL},
	}

	acc := &testutil.Accumulator{}
	plugin.Gather(acc)
}

//func TestLogstash6(t *testing.T) {
//	s := getHTTPServerVersion("logstash_6")
//	defer s.Close()
//
//	plugin := &logstash{
//		Servers: []string{s.URL},
//	}
//
//	acc := &testutil.Accumulator{}
//	plugin.Gather(acc)
//}