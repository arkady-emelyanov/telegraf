package logstash

import (
	"fmt"
	"net/url"
	"net/http"
	"time"
	"sync"

	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf"
	"reflect"
)

const defaultAPIPrefix = "/_/node/stats"
const configSample = `
 ## Hello
 #gather_types = [ "jvm", "process", "events", "pipeline" ]
`

type (
	// burrow plugin
	logstash struct {
		Servers []string

		Username string
		Password string
		Timeout  internal.Duration

		APIPrefix string `toml:"api_prefix"`

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
	return "Collect Logstash events via Logstash Node API."
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
			go publishEventsStat(api, res, &workers)
			go publishJVMStat(api, res, &workers)
			go publishProcessStat(api, res, &workers)
			go publishPipelinesStat(api, res, &workers)
		})

		endpointChan <- c.apiPrefix
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
		l.APIPrefix = defaultAPIPrefix
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

// worker spawn helper function
func withAPICall(api apiClient, producer <-chan string, done chan<- bool, resolver resolverFn) {
	for {
		uri := <-producer
		if uri == "" {
			break
		}

		fmt.Println("request:", uri)
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
