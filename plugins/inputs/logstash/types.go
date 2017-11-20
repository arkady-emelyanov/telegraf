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
		Host        string `json:"host"`
		Version     string `json:"version"`
		HTTPAddress string `json:"http_address"`
		ID          string `json:"id"`
		Name        string `json:"name"`

		// possible keys
		Process   apiProcessResponse  `json:"process"`
		JVM       apiJVMResponse      `json:"jvm"`
		Pipeline  apiPipelineResponse `json:"pipeline"` // logstash 5.x
		Pipelines apiPipelineResponse `json:"pipelines"` // logstash 6.x
	}

	apiProcessResponse struct {
		OpenFileDescriptors     int `json:"open_file_descriptors"`
		PeakOpenFileDescriptors int `json:"peak_open_file_descriptors"`
		MaxFileDescriptors      int `json:"max_file_descriptors"`

		Mem apiProcessResponseMem `json:"mem"`
		CPU apiProcessResponseCPU `json:"cpu"`
	}

	apiProcessResponseMem struct {
		TotalVirtualInBytes int64 `json:"total_virtual_in_bytes"`
	}

	apiProcessResponseCPU struct {
		TotalInMillis int64                            `json:"total_in_millis"`
		Percent       float32                          `json:"percent"`
		LoadAverage   apiProcessResponseCPULoadAverage `json:"load_average"`
	}

	apiProcessResponseCPULoadAverage struct {
		OneMinute      float32 `json:"1m"`  //"1m": 0.23,
		FiveMinutes    float32 `json:"5m"`  //"5m": 0.35,
		FifteenMinutes float32 `json:"15m"` //"15m": 0.27
	}

	apiJVMResponse struct {
		Uptime int64 `json:"uptime_in_millis"`

		Threads apiJVMResponseThreads `json:"threads"`
		Mem     apiJVMResponseMem     `json:"mem"`
		GC      apiJVMResponseGC      `json:"gc"`
	}

	apiJVMResponseThreads struct {
		Count     int `json:"count"`
		PeakCount int `json:"peak_count"`
	}

	apiJVMResponseMem struct {
		HeapUserPercent        float32 `json:"heap_user_percent"`
		HeapCommittedInBytes   int64   `json:"heap_committed_in_bytes"`
		HeapMaxInBytes         int64   `json:"heap_max_in_bytes"`
		HeapUsedInBytes        int64   `json:"heap_used_in_bytes"`
		NonHeapUsedInBytes     int64   `json:"non_heap_used_in_bytes"`
		NonHeapCommitedInBytes int64   `json:"non_heap_commited_in_bytes"`

		Pools apiJVMResponseMemPools `json:"pools"`
	}

	apiJVMResponseMemPools struct {
		Survivor apiJVMResponseMemPoolItem `json:"survivor"`
		Old      apiJVMResponseMemPoolItem `json:"old"`
		Young    apiJVMResponseMemPoolItem `json:"young"`
	}

	apiJVMResponseMemPoolItem struct {
		PeakUsedInBytes  int64 `json:"peak_used_in_bytes"`
		UsedInBytes      int64 `json:"used_in_bytes"`
		PeakMaxInBytes   int64 `json:"peak_max_in_bytes"`
		MaxInBytes       int64 `json:"max_in_bytes"`
		CommittedInBytes int64 `json:"committed_in_bytes"`
	}

	apiJVMResponseGC struct {
		Collectors apiJVMResponseGCCollector `json:"collectors"`
	}

	apiJVMResponseGCCollector struct {
		Old   apiJVMResponseGCCollectorItem `json:"old"`
		Young apiJVMResponseGCCollectorItem `json:"young"`
	}

	apiJVMResponseGCCollectorItem struct {
		CollectionTimeInMillis int64 `json:"collection_time_in_millis"`
		CollectionCount        int   `json:"collection_count"`
	}

	apiPipelineResponse struct {
		ID string `json:"id"`

		Events  apiPipelineResponseEvents  `json:"events"`
		Plugins apiPipelineResponsePlugins `json:"plugins"`
		Queue   apiPipelineResponseQueue   `json:"queue"`
	}

	apiPipelineResponseEvents struct {
		DurationInMillis          int64 `json:"duration_in_millis"`
		In                        int64 `json:"in"`
		Out                       int64 `json:"out"`
		Filtered                  int64 `json:"filtered"`
		QueuePushDurationInMillis int64 `json:"queue_push_duration_in_millis"`
	}

	apiPipelineResponsePlugins struct {
		Inputs  []apiPipelineResponsePluginInput  `json:"inputs"`
		Filters []apiPipelineResponsePluginFilter `json:"filters"`
		Outputs []apiPipelineResponsePluginOutput `json:"outputs"`
	}

	// /pipeline/plugins/input
	apiPipelineResponsePluginInput struct {
		ID   string `json:"id"`
		Name string `json:"name"`

		Events apiPipelineResponsePluginInputEvents `json:"events"`
	}

	apiPipelineResponsePluginInputEvents struct {
		QueuePushDurationInMillis int64 `json:"queue_push_duration_in_millis"`
		Out                       int64 `json:"out"`
	}

	// /pipeline/plugins/filter
	apiPipelineResponsePluginFilter struct {
		ID   string `json:"id"`
		Name string `json:"name"`

		Events apiPipelineResponsePluginFilterEvents `json:"events"`
	}

	apiPipelineResponsePluginFilterEvents struct {
	}

	// /pipeline/plugins/output
	apiPipelineResponsePluginOutput struct {
		ID   string `json:"id"`
		Name string `json:"name"`

		Events apiPipelineResponsePluginOutputEvents `json:"events"`
	}

	apiPipelineResponsePluginOutputEvents struct {
		DurationInMillis int64 `json:"duration_in_millis"`
		In               int64 `json:"in"`
		Out              int64 `json:"out"`
	}

	apiPipelineResponseQueue struct {
		Type string `json:"type"`
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
