package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func ParseResponse[T any](res *esapi.Response) ([]T, error) {
	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"` // Use map for flexible field handling
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	var items []T
	for _, hit := range result.Hits.Hits {
		// Normalize fields before unmarshalling into the target struct
		normalizeNumericFields(hit.Source)

		// Marshal the normalized map back to JSON
		data, err := json.Marshal(hit.Source)
		if err != nil {
			log.Printf("Error marshaling normalized data: %v", err)
			continue
		}

		// Unmarshal into the target struct
		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			log.Printf("Error unmarshalling hit: %v", err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func normalizeNumericFields(data map[string]interface{}) {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}: // Recurse for nested fields
			normalizeNumericFields(v)
		case []interface{}: // Handle arrays of nested fields
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					normalizeNumericFields(m)
				}
			}
		case string: // Convert numeric strings to float64
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				data[key] = f
			}
		}
	}
}

// MapToReader converts a map to a JSON reader for Elasticsearch queries.
func MapToReader(query map[string]interface{}) (*bytes.Reader, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %v", err)
	}
	return bytes.NewReader(body), nil
}

// // ParseResponse parses an Elasticsearch response into a slice of the specified type.
// func ParseResponse[T any](res *esapi.Response) ([]T, error) {
// 	var result struct {
// 		Hits struct {
// 			Hits []struct {
// 				Source json.RawMessage `json:"_source"`
// 			} `json:"hits"`
// 		} `json:"hits"`
// 	}

// 	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
// 		return nil, fmt.Errorf("failed to decode response body: %v", err)
// 	}

// 	var items []T
// 	for _, hit := range result.Hits.Hits {
// 		var item T
// 		if err := json.Unmarshal(hit.Source, &item); err != nil {
// 			log.Printf("Error unmarshalling hit: %v", err)
// 			continue
// 		}
// 		items = append(items, item)
// 	}

// 	return items, nil
// }

// // MapToReader converts a map to a JSON reader for Elasticsearch queries.
// func MapToReader(query map[string]interface{}) (*bytes.Reader, error) {
// 	body, err := json.Marshal(query)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal query: %v", err)
// 	}
// 	return bytes.NewReader(body), nil
// }
