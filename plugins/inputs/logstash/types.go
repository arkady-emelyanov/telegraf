package logstash

import (
	"net/http"
	"encoding/json"
	"fmt"

	"github.com/influxdata/telegraf"
)

type (
	apiClient struct {
		client      http.Client
		acc         telegraf.Accumulator
		apiPrefix   string
		baseURL     string
		requestUser string
		requestPass string
		errorChan   chan<- error
	}

	apiResponse struct {
		// general request
		Host        string
		Version     string
		HTTPAddress string
		ID          string
		Name        string

		// possible keys
		JVM     apiJVMResponse
		Process apiProcessResponse
	}

	apiProcessResponse struct {
		//"open_file_descriptors": 116,
		//"peak_open_file_descriptors": 117,
		//"max_file_descriptors": 1048576,
		Mem apiProcessResponseMem
	}

	apiProcessResponseMem struct {
		//"total_virtual_in_bytes": 4848947200
	}

	apiProcessResponseCPU struct {
		//"total_in_millis": 48300,
		//"percent": 1,
	}

	apiProcessResponseCPULoadAverage struct {
		//"1m": 0.23,
		//"5m": 0.35,
		//"15m": 0.27
	}

	apiJVMResponse struct {
		Threads apiJVMResponseThreads
		Mem     apiJVMResponseMem
		GC      apiJVMResponseGC
		Uptime  int64 `json:"uptime_in_millis"`
	}

	apiJVMResponseThreads struct {
		Count     int
		PeakCount int
	}

	apiJVMResponseMem struct {
		//"heap_used_percent": 19,
		//"heap_committed_in_bytes": 1038876672,
		//"heap_max_in_bytes": 1038876672,
		//"heap_used_in_bytes": 207075480,
		//"non_heap_used_in_bytes": 94490168,
		//"non_heap_committed_in_bytes": 100573184,
		Pools apiJVMResponseMemPools
	}

	apiJVMResponseMemPools struct {
		Survivor apiJVMResponseMemPoolItem
		Old      apiJVMResponseMemPoolItem
		Young    apiJVMResponseMemPoolItem
	}

	apiJVMResponseMemPoolItem struct {
		//"peak_used_in_bytes": 34865152,
		//"used_in_bytes": 34865144,
		//"peak_max_in_bytes": 34865152,
		//"max_in_bytes": 34865152,
		//"committed_in_bytes": 34865152
	}

	apiJVMResponseGC struct {
		Collectors apiJVMResponseGCCollector
	}

	apiJVMResponseGCCollector struct {
		Old   apiJVMResponseGCCollectorItem
		Young apiJVMResponseGCCollectorItem
	}

	apiJVMResponseGCCollectorItem struct {
		//"collection_time_in_millis": 182,
		//"collection_count": 2
	}
)


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
