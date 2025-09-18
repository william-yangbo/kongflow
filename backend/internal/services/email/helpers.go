package email

import "encoding/json"

// MarshalEmailData is a helper function to marshal email data
func MarshalEmailData(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}