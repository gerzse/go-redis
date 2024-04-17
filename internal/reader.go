package internal

// Reader allows Cmders to read their responses, regardless of the underlying transport.
type Reader interface {
	Peek(n int) ([]byte, error)
	PeekReplyType() (byte, error)
	ReadLine() ([]byte, error)
	ReadReply() (interface{}, error)
	ReadInt() (int64, error)
	ReadUint() (uint64, error)
	ReadFloat() (float64, error)
	ReadString() (string, error)
	ReadBool() (bool, error)
	ReadSlice() ([]interface{}, error)
	ReadFixedArrayLen(fixedLen int) error
	ReadArrayLen() (int, error)
	ReadFixedMapLen(fixedLen int) error
	ReadMapLen() (int, error)
	DiscardNext() error
}
