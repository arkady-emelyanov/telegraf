package logstash

import (
	"sync"
)

func publishProcessStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()
}
