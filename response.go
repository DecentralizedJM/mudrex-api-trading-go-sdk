package mudrex

import (
	"encoding/json"
	"fmt"
)

// Response wraps API object payloads and supports map-style field access.
type Response map[string]json.RawMessage

// Get unmarshals a field into dst.
func (r Response) Get(key string, dst any) error {
	raw, ok := r[key]
	if !ok {
		return fmt.Errorf("response has no field %q", key)
	}
	return json.Unmarshal(raw, dst)
}

// GetString returns a string field value.
func (r Response) GetString(key string) (string, error) {
	var value string
	if err := r.Get(key, &value); err != nil {
		return "", err
	}
	return value, nil
}

// GetBool returns a bool field value.
func (r Response) GetBool(key string) (bool, error) {
	var value bool
	if err := r.Get(key, &value); err != nil {
		return false, err
	}
	return value, nil
}

// Result returns the scalar result field when the API returns a non-object value.
func (r Response) Result() (json.RawMessage, bool) {
	raw, ok := r["result"]
	return raw, ok
}

func responseFromMap(data map[string]any) (Response, error) {
	out := make(Response, len(data))
	for key, value := range data {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		out[key] = raw
	}
	return out, nil
}
