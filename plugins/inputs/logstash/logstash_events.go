package logstash

import (
	"fmt"
	"sync"
)

func publishEventStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println(res)
}
