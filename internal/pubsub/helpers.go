package pubsub

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func encodeGob[T any](val T) ([]byte, error) {
	var buf bytes.Buffer

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(val)
	if err != nil {
		return nil, fmt.Errorf("failed to encode: %v", err)
	}

	return buf.Bytes(), nil
}

func decodeGob[T any](data []byte) (T, error) {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var val T
	err := decoder.Decode(&val)
	if err != nil {
		return val, fmt.Errorf("error decoding bytes: %v", err)
	}

	return val, nil
}
