package logstash

import (
	"sync"
)

func publishJVMStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()
}
