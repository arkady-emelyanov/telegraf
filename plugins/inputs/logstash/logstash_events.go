package logstash

import (
	"fmt"
	"sync"
	"strings"
)

func publishEventsStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	// events not supported before 6.x version
	if strings.HasPrefix(res.Version, "5.") {
		return
	}

	// publish events
	fmt.Println("events:", res)
}
