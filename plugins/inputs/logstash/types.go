package logstash

import (
	"net/http"

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
