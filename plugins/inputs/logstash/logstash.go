package logstash

import (
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf"
	"fmt"
	"net/url"
	"net/http"
	"time"
	"encoding/json"
	"sync"
	"strings"
)

const configSample = `
 ## Hello
`

type (
	// burrow plugin
	logstash struct {
		Servers []string

		Username string
		Password string
		Timeout  internal.Duration

		APIPrefix string `toml:"api_prefix"`
		Types     []string

		// Path to CA file
		SSLCA string `toml:"ssl_ca"`
		// Path to host cert file
		SSLCert string `toml:"ssl_cert"`
		// Path to cert key file
		SSLKey string `toml:"ssl_key"`
		// Use SSL but skip chain & host verification
		InsecureSkipVerify bool
	}

	// function prototype for worker spawning helper
	resolverFn func(api apiClient, res apiResponse)
)

func init() {
	inputs.Add("logstash", func() telegraf.Input {
		return &logstash{}
	})
}

func (l *logstash) SampleConfig() string {
	return configSample
}

func (l *logstash) Description() string {
	return "Collect Kafka topics and consumers status from Burrow HTTP API."
}

// Gather Burrow stats
func (l *logstash) Gather(acc telegraf.Accumulator) error {
	var workers sync.WaitGroup

	errorChan := l.getErrorChannel(acc)
	for _, addr := range l.Servers {
		c, err := l.getClient(acc, addr, errorChan)
		if err != nil {
			errorChan <- err
			continue
		}

		endpointChan := make(chan string)
		workers.Add(4)

		go withAPICall(c, endpointChan, nil, func(api apiClient, res apiResponse) {
			go publishEventStat(api, res, &workers)
			go publishJVMStat(api, res, &workers)
			go publishProcessStat(api, res, &workers)
			go publishPipelineStat(api, res, &workers)
		})

		collectTypes := strings.Join(l.Types, ",")
		if collectTypes == "" {
			collectTypes = "jvm,process,pipelines,events"
		}

		endpointChan <- fmt.Sprintf("%s/%s", c.apiPrefix, collectTypes)
		close(endpointChan)
	}

	workers.Wait()
	close(errorChan)

	return nil
}

// Error collector / register
func (l *logstash) getErrorChannel(acc telegraf.Accumulator) chan error {
	errorChan := make(chan error)
	go func(acc telegraf.Accumulator) {
		for {
			err := <-errorChan
			if err != nil {
				acc.AddError(err)
			} else {
				break
			}
		}
	}(acc)

	return errorChan
}

// API client construction
func (l *logstash) getClient(acc telegraf.Accumulator, addr string, errorChan chan<- error) (apiClient, error) {
	var c apiClient

	u, err := url.Parse(addr)
	if err != nil {
		return c, err
	}

	// Override global configuration (if specified in url)
	requestUser := l.Username
	requestPass := l.Password
	if u.User != nil {
		requestUser = u.User.Username()
		requestPass, _ = u.User.Password()
	}

	// Enable SSL configuration (if provided)
	tlsCfg, err := internal.GetTLSConfig(l.SSLCert, l.SSLKey, l.SSLCA, l.InsecureSkipVerify)
	if err != nil {
		return c, err
	}

	// api prefix
	if l.APIPrefix == "" {
		l.APIPrefix = "/_/node/stats"
	}

	if l.Timeout.Duration < time.Second {
		l.Timeout.Duration = time.Second * 5
	}

	// create client
	c = apiClient{
		client: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
			Timeout: l.Timeout.Duration,
		},
		acc:         acc,
		baseURL:     fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		apiPrefix:   l.APIPrefix,
		requestUser: requestUser,
		requestPass: requestPass,
		errorChan:   errorChan,
	}

	return c, nil
}

// construct endpoint request
func (api *apiClient) getRequest(uri string) (*http.Request, error) {
	// create new request
	endpoint := fmt.Sprintf("%s%s", api.baseURL, uri)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// add support for http basic authorization
	if api.requestUser != "" {
		req.SetBasicAuth(api.requestUser, api.requestPass)
	}

	return req, nil
}

// perform synchronous http request
func (api *apiClient) call(uri string) (apiResponse, error) {
	var r apiResponse

	// get request
	req, err := api.getRequest(uri)
	if err != nil {
		return r, err
	}

	// do request
	res, err := api.client.Do(req)
	if err != nil {
		return r, err
	}

	// decode response
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return r, fmt.Errorf("endpoint: '%s', invalid response code: '%d'", uri, res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return r, err
	}

	return r, err
}

// worker spawn helper function
func withAPICall(api apiClient, producer <-chan string, done chan<- bool, resolver resolverFn) {
	for {
		uri := <-producer
		if uri == "" {
			break
		}

		fmt.Println(uri)
		res, err := api.call(uri)
		if err != nil {
			api.errorChan <- err
		}

		resolver(api, res)
		if done != nil {
			done <- true
		}
	}
}
