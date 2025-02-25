package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hoyle1974/khronoscope/internal/resources"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

const RESOURCE_INDEX = "k8s_resources"

type esImpl struct {
	es *elasticsearch.Client
}

func NewESSource() KhronoStore {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Error creating Elasticsearch client")
	}

	return &esImpl{es: es}
}

func (d *esImpl) Size() int            { return -1 }
func (d *esImpl) Save(filename string) {}

func (d *esImpl) SetLabel(time time.Time, label string) {
}

func (d *esImpl) GetLabel(time time.Time) string {
	return ""
}

func (d *esImpl) GetNextLabelTime(time time.Time) time.Time {
	return time
}

func (d *esImpl) GetPrevLabelTime(time time.Time) time.Time {
	return time
}

func (d *esImpl) GetTimeRange() (time.Time, time.Time) {
	a, b, err := getMinMaxTimestamps(d.es, RESOURCE_INDEX)
	if err != nil {
		return a, b
	}
	return time.Now(), time.Now()
}

func getMinMaxTimestamps(es *elasticsearch.Client, indexName string) (minTime time.Time, maxTime time.Time, err error) {
	query := map[string]interface{}{
		"aggs": map[string]interface{}{
			"min_timestamp": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "timestamp",
				},
			},
			"max_timestamp": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "timestamp",
				},
			},
		},
		"size": 0,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return minTime, maxTime, fmt.Errorf("error encoding query: %v", err)
	}

	res, err := es.Search(
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
	)
	if err != nil {
		return minTime, maxTime, fmt.Errorf("error executing search: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return minTime, maxTime, fmt.Errorf("error in search response: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return minTime, maxTime, fmt.Errorf("error decoding response body: %v", err)
	}

	aggs := r["aggregations"].(map[string]interface{})
	minTimestamp, ok := aggs["min_timestamp"].(map[string]interface{})
	if !ok {
		return minTime, maxTime, fmt.Errorf("min_timestamp not found or not a map")
	}
	minTimestampStr, ok := minTimestamp["value_as_string"].(string)
	if !ok {
		return minTime, maxTime, fmt.Errorf("min_timestamp value_as_string not found or not a string")
	}
	maxTimestamp, ok := aggs["max_timestamp"].(map[string]interface{})
	if !ok {
		return minTime, maxTime, fmt.Errorf("max_timestamp not found or not a map")
	}
	maxTimestampStr, ok := maxTimestamp["value_as_string"].(string)
	if !ok {
		return minTime, maxTime, fmt.Errorf("max_timestamp value_as_string not found or not a string")
	}

	minTime, err = time.Parse(time.RFC3339Nano, minTimestampStr)
	if err != nil {
		return minTime, maxTime, fmt.Errorf("error parsing min timestamp: %v", err)
	}

	maxTime, err = time.Parse(time.RFC3339Nano, maxTimestampStr)
	if err != nil {
		return minTime, maxTime, fmt.Errorf("error parsing max timestamp: %v", err)
	}

	return minTime, maxTime, nil
}

func getDocumentId(resource resources.Resource) string {
	return fmt.Sprintf("%s-%s", resource.Uid, resource.Timestamp.Time.Format(time.RFC3339Nano))
}

type ESResource struct {
	Timestamp time.Time
	Uid       string
	Resource  resources.Resource
	Action    string
}

func (d *esImpl) indexResource(action string, resource resources.Resource) {
	esr := ESResource{
		Timestamp: resource.Timestamp.Time,
		Uid:       resource.Uid,
		Action:    action,
		Resource:  resource,
	}

	docJSON, err := json.Marshal(esr)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Error marshaling document to JSON")
	}

	req := esapi.IndexRequest{
		Index:      RESOURCE_INDEX,          // Replace with your index name
		DocumentID: getDocumentId(resource), // Replace with your document ID (optional)
		Body:       bytes.NewReader(docJSON),
		Refresh:    "true", // Makes the document immediately searchable
	}

	// Context to be used by the watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := req.Do(ctx, d.es)
	if err != nil {
		log.Fatal().Stack().Err(err).Stack().Msg("Error indexing document")
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Fatal().Err(err).Any("JSON", docJSON).Str("Result", res.String()).Msg("Error indexing document")
	}
}

func (d *esImpl) AddResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Add")
	d.indexResource("add", resource)
}

func (d *esImpl) UpdateResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Update")
	d.indexResource("update", resource)
}

func (d *esImpl) DeleteResource(resource resources.Resource) {
	log.Debug().Str("Resource", resource.Name).Str("Uid", resource.Uid).Msg("Delete")
	d.indexResource("delete", resource)
}

func (d *esImpl) GetResourceAt(timestamp time.Time, uid string) (resources.Resource, error) {
	for _, r := range d.GetResourcesAt(timestamp, "", "") {
		if r.Uid == uid {
			return r, nil
		}
	}

	return resources.Resource{}, nil
}

func (d *esImpl) GetResourcesAt(timestamp time.Time, kind string, namespace string) []resources.Resource {
	ret := []resources.Resource{}

	if esr, err := getLatestResources(d.es, RESOURCE_INDEX, timestamp); err != nil {
		for _, e := range esr {
			if e.Action != "delete" {
				ret = append(ret, e.Resource)
			}
		}
	}

	return ret
}

func getLatestResources(es *elasticsearch.Client, indexName string, queryTimestamp time.Time) ([]ESResource, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": map[string]interface{}{
					"range": map[string]interface{}{
						"timestamp": map[string]interface{}{
							"lte": queryTimestamp.Format(time.RFC3339Nano),
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"latest_resources": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "uid.keyword",
					"size":  1000, // Adjust as needed.
				},
				"aggs": map[string]interface{}{
					"latest": map[string]interface{}{
						"top_hits": map[string]interface{}{
							"size": 1,
							"sort": []map[string]interface{}{
								{"timestamp": map[string]interface{}{"order": "desc"}},
							},
						},
					},
				},
			},
		},
		"size": 0,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("error encoding query: %v", err)
	}

	res, err := es.Search(
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
	)

	if err != nil {
		return nil, fmt.Errorf("error executing search: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error in search response: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	buckets := r["aggregations"].(map[string]interface{})["latest_resources"].(map[string]interface{})["buckets"].([]interface{})
	var latestResources []ESResource
	for _, bucket := range buckets {
		hits := bucket.(map[string]interface{})["latest"].(map[string]interface{})["hits"].(map[string]interface{})["hits"].([]interface{})
		if len(hits) > 0 {
			source := hits[0].(map[string]interface{})["_source"]
			data, err := json.Marshal(source)
			if err != nil {
				log.Printf("Error marshaling source: %v", err)
				continue
			}
			var esr ESResource
			if err := json.Unmarshal(data, &esr); err != nil {
				log.Printf("Error unmarshaling resource: %v", err)
				continue
			}
			latestResources = append(latestResources, esr)
		}
	}
	return latestResources, nil
}
