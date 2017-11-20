package logstash

import (
	"fmt"
	"sync"
)

func publishJVMStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("jvm:", res)
}
