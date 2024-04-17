package cache

import (
	"github.com/redis/go-redis/v9/internal"
)

// ReaderSpy wraps a Reader and collects all data that is being read.
// The collected data can be stored in the cache, and later replayed.
type ReaderSpy struct {
	cache *Cache
	data  []interface{}
}

func (ri *ReaderSpy) StoreInCache(key interface{}) (bool, error) {
	return ri.cache.setKey(key, ri.data)
}

func (ri *ReaderSpy) wrapReader(reader internal.Reader) *bufferingReaderWrapper {
	return &bufferingReaderWrapper{
		reader: reader,
		data:   &ri.data,
	}
}

// bufferingReaderWrapper is a one-off wrapper around a Reader, backed by the
// slice in a ReaderSpy. It is needed because we don't have access to the creation
// of actual Readers, but only to read functions where the already created readers
// are sent as parameters, so we must accumulate data from several such calls.
type bufferingReaderWrapper struct {
	reader internal.Reader
	data   *[]interface{}
}

func (cr *bufferingReaderWrapper) appendData(dataItem interface{}) {
	*cr.data = append(*cr.data, dataItem)
}

func (cr *bufferingReaderWrapper) Peek(n int) ([]byte, error) {
	result, err := cr.reader.Peek(n)
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) PeekReplyType() (byte, error) {
	result, err := cr.reader.PeekReplyType()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadLine() ([]byte, error) {
	result, err := cr.reader.ReadLine()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadReply() (interface{}, error) {
	result, err := cr.reader.ReadReply()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadInt() (int64, error) {
	result, err := cr.reader.ReadInt()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadUint() (uint64, error) {
	result, err := cr.reader.ReadUint()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadFloat() (float64, error) {
	result, err := cr.reader.ReadFloat()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadString() (string, error) {
	s, err := cr.reader.ReadString()
	*cr.data = append(*cr.data, s)
	return s, err
}

func (cr *bufferingReaderWrapper) ReadBool() (bool, error) {
	result, err := cr.reader.ReadBool()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadSlice() ([]interface{}, error) {
	result, err := cr.reader.ReadSlice()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadFixedArrayLen(fixedLen int) error {
	return cr.reader.ReadFixedArrayLen(fixedLen)
}

func (cr *bufferingReaderWrapper) ReadArrayLen() (int, error) {
	result, err := cr.reader.ReadArrayLen()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) ReadFixedMapLen(fixedLen int) error {
	return cr.reader.ReadFixedMapLen(fixedLen)
}

func (cr *bufferingReaderWrapper) ReadMapLen() (int, error) {
	result, err := cr.reader.ReadMapLen()
	cr.appendData(result)
	return result, err
}

func (cr *bufferingReaderWrapper) DiscardNext() error {
	return cr.reader.DiscardNext()
}
