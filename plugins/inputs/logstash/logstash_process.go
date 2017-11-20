package logstash

import (
	"fmt"
	"sync"
)

func publishProcessStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("process:", res)
}
