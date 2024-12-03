package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ParseResponse parses an Elasticsearch response into a slice of the specified type.
func ParseResponse[T any](res *esapi.Response) ([]T, error) {
	var result struct {
		Hits struct {
			Hits []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	var items []T
	for _, hit := range result.Hits.Hits {
		var item T
		if err := json.Unmarshal(hit.Source, &item); err != nil {
			log.Printf("Error unmarshalling hit: %v", err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// MapToReader converts a map to a JSON reader for Elasticsearch queries.
func MapToReader(query map[string]interface{}) (*bytes.Reader, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %v", err)
	}
	return bytes.NewReader(body), nil
}
