package proxy

import "encoding/json"

func InjectStreamOptions(body []byte) ([]byte, bool, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return body, false, err
	}
	stream, ok := req["stream"].(bool)
	if !ok || !stream {
		return body, false, nil
	}
	if so, exists := req["stream_options"]; exists && so != nil {
		return body, false, nil
	}
	req["stream_options"] = map[string]interface{}{
		"include_usage": true,
	}
	modified, err := json.Marshal(req)
	if err != nil {
		return body, false, err
	}
	return modified, true, nil
}
