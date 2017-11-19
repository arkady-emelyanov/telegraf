package logstash

import (
	"fmt"
	"sync"
)

func publishPipelineStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println(res)
}
