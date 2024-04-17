package cache

import (
	"errors"
	"fmt"
	"reflect"
)

type ReaderReplayer struct {
	data []interface{}
}

func popFirst[T any](s *[]interface{}) (T, error) {
	var zeroValue T

	if len(*s) == 0 {
		return zeroValue, errors.New("no more data available in the cached value")
	}

	first, ok := (*s)[0].(T)
	if !ok {
		return zeroValue, fmt.Errorf("type mismatch in cached value, expected %s, got %T", reflect.TypeOf(zeroValue).Name(), (*s)[0])
	}

	*s = (*s)[1:]
	return first, nil
}

func (r *ReaderReplayer) Peek(n int) ([]byte, error) {
	return popFirst[[]byte](&r.data)
}

func (r *ReaderReplayer) PeekReplyType() (byte, error) {
	return popFirst[byte](&r.data)
}

func (r *ReaderReplayer) ReadLine() ([]byte, error) {
	return popFirst[[]byte](&r.data)
}

func (r *ReaderReplayer) ReadReply() (interface{}, error) {
	return popFirst[interface{}](&r.data)
}

func (r *ReaderReplayer) ReadInt() (int64, error) {
	return popFirst[int64](&r.data)
}

func (r *ReaderReplayer) ReadUint() (uint64, error) {
	return popFirst[uint64](&r.data)
}

func (r *ReaderReplayer) ReadFloat() (float64, error) {
	return popFirst[float64](&r.data)
}

func (r *ReaderReplayer) ReadString() (string, error) {
	return popFirst[string](&r.data)
}

func (r *ReaderReplayer) ReadBool() (bool, error) {
	return popFirst[bool](&r.data)
}

func (r *ReaderReplayer) ReadSlice() ([]interface{}, error) {
	return popFirst[[]interface{}](&r.data)
}

func (r *ReaderReplayer) ReadFixedArrayLen(int) error {
	return nil
}

func (r *ReaderReplayer) ReadArrayLen() (int, error) {
	return popFirst[int](&r.data)
}

func (r *ReaderReplayer) ReadFixedMapLen(int) error {
	return nil
}

func (r *ReaderReplayer) ReadMapLen() (int, error) {
	return popFirst[int](&r.data)
}

func (r *ReaderReplayer) DiscardNext() error {
	return nil
}
