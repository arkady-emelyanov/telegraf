package logstash

import (
	"fmt"
	"sync"
)

func publishPipelinesStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("pipeline:", res)
}
