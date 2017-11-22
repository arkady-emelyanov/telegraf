package logstash

import (
	"fmt"
	"sync"
	"strings"
)

func publishPipelinesStat(api apiClient, res apiResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	pipeline := res.Pipelines
	if strings.HasPrefix(res.Version, "5.") {
		pipeline = res.Pipeline
	}

	publishPipelineEvents(api, pipeline.Events)
	publishPipelinePlugin(api, "inputs", pipeline.Plugins.Inputs)
	publishPipelinePlugin(api, "filters", pipeline.Plugins.Filters)
	publishPipelinePlugin(api, "outputs", pipeline.Plugins.Outputs)
}

func publishPipelineEvents(api apiClient, events apiPipelineResponseEvents) {
	fmt.Println("pipeline events:", events)
}

func publishPipelinePlugin(api apiClient, t string, plugins []map[string]interface{}) {
	for _, p := range plugins {
		if _, ok := p["id"]; !ok {
			continue
		}
		if _, ok := p["events"]; !ok {
			continue
		}
		if _, ok := p["name"]; !ok {
			continue
		}

		//pluginId, _ := p["id"]
		//pluginName, _ := p["name"]
		eventsRaw, _ := p["events"]

		events, ok := eventsRaw.(map[string]interface{})
		if !ok {
			continue
		}

		delete(p, "id")
		delete(p, "events")
		delete(p, "name")

		fields := make([]string, 0)
		for k, v := range events {
			val, ok := v.(float64)
			if !ok {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s=%d", k, int64(val)))
		}

		for k, v := range p {
			// v can be map[string]interface{}
			// v can be float64
			// v can be int64
			fmt.Println(k, v)
		}

		fmt.Println(strings.Join(fields,","))
	}
}
