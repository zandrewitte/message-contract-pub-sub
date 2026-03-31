package encoding

import "encoding/json"

type MessageEncoder interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}

type jsonEncoder struct{}

func (j *jsonEncoder) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j *jsonEncoder) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func NewJSONEncoder() MessageEncoder {
	return &jsonEncoder{}
}
