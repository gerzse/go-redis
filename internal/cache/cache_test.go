package cache_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/bsm/ginkgo/v2"
	. "github.com/bsm/gomega"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/internal"
	"github.com/redis/go-redis/v9/internal/cache"
)

func TestSetCache(t *testing.T) {
	cacheInstance, err := cache.NewCache(&cache.Config{MaxKeys: 1000, MaxSize: 1 << 20})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: ":6379", CacheObject: cacheInstance})
	defer client.Close()
	ctx := context.Background()
	client.Ping(ctx)

	metrics := cacheInstance.Metrics()
	keysAdded := metrics.KeysAdded()
	t.Log(keysAdded)

	val, found := waitGet(client.Options().CacheObject, []interface{}{"ping"})
	if found {
		t.Log(val)
	} else {
		t.Error("Key not found")
	}
	ping := client.Ping(ctx)
	t.Log(ping.Val())
	if ping.Val() == "PONG" {
		t.Log(ping.Val())
	} else {
		t.Error("Ping from cache failed")
	}
}

func TestClientTracking(t *testing.T) {
	ctx := context.Background()

	clientA := redis.NewClient(&redis.Options{
		Addr:           ":6379",
		Protocol:       3,
		MaxActiveConns: 1,
		PoolSize:       1,
	})
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			t.Error(err)
		}
	}(clientA)

	clientB := redis.NewClient(&redis.Options{
		Addr:           ":6379",
		Protocol:       3,
		MaxActiveConns: 1,
		PoolSize:       1,
	})
	defer func(clientB *redis.Client) {
		err := clientB.Close()
		if err != nil {
			t.Error(err)
		}
	}(clientB)

	clientA.FlushAll(ctx)

	_, err := clientA.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ClientTrackingOn(ctx)
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	_, err = clientA.Set(ctx, "key", "initial value", 0).Result()
	if err != nil {
		t.Error(err)
	}

	var initialValue string
	initialValue, err = clientA.Get(ctx, "key").Result()
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Initial value in clientA:", initialValue)

	_, err = clientB.Set(ctx, "key", "updated value", 0).Result()
	if err != nil {
		t.Error(err)
	}

	// Wait to ensure the invalidation message is received
	time.Sleep(5 * time.Second)

	for i := 0; i < 5; i += 1 {
		var updatedValue string
		updatedValue, err = clientA.Get(ctx, "key").Result()
		if err != nil {
			t.Error(err)
		}
		fmt.Println("Updated value in clientA:", updatedValue)
	}

	_, err = clientA.Set(ctx, "key", "Over!", 0).Result()
	if err != nil {
		t.Error(err)
	}
}

func TestGinkgoSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "expected type")
}

var _ = Describe("Cache", func() {
	var (
		mockReader    *MockInternalReader
		cacheInstance *cache.Cache
	)

	BeforeEach(func() {
		mockReader = NewMockInternalReader()

		var err error
		cacheInstance, err = cache.NewCache(&cache.Config{MaxKeys: 1000, MaxSize: 1 << 20})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle peek", func() {
		runWithValue(cacheInstance, mockReader, []byte("aaaaa"), func(rd internal.Reader) ([]byte, error) {
			return rd.Peek(5)
		})
	})

	It("should handle peek reply type", func() {
		runWithValue(cacheInstance, mockReader, 'x', func(rd internal.Reader) (byte, error) {
			return rd.PeekReplyType()
		})
	})

	It("should handle line", func() {
		runWithValue(cacheInstance, mockReader, []byte("line"), func(rd internal.Reader) ([]byte, error) {
			return rd.ReadLine()
		})
	})

	It("should handle reply", func() {
		runWithValue(cacheInstance, mockReader, "reply", func(rd internal.Reader) (interface{}, error) {
			return rd.ReadReply()
		})
	})

	It("should handle int64", func() {
		runWithValue(cacheInstance, mockReader, 42, func(rd internal.Reader) (int64, error) {
			return rd.ReadInt()
		})
	})

	It("should handle uint64", func() {
		runWithValue(cacheInstance, mockReader, uint64(42), func(rd internal.Reader) (uint64, error) {
			return rd.ReadUint()
		})
	})

	It("should handle float", func() {
		runWithValue(cacheInstance, mockReader, 3.14, func(rd internal.Reader) (float64, error) {
			return rd.ReadFloat()
		})
	})

	It("should handle string", func() {
		runWithValue(cacheInstance, mockReader, "foo", func(rd internal.Reader) (string, error) {
			return rd.ReadString()
		})
	})

	It("should handle bool", func() {
		runWithValue(cacheInstance, mockReader, true, func(rd internal.Reader) (bool, error) {
			return rd.ReadBool()
		})
	})

	It("should handle slice", func() {
		runWithValue(cacheInstance, mockReader, []interface{}{"slice"}, func(rd internal.Reader) ([]interface{}, error) {
			return rd.ReadSlice()
		})
	})

	It("should handle fixed array len", func() {
		runWithoutValue(cacheInstance, mockReader, func(rd internal.Reader) error {
			return rd.ReadFixedArrayLen(1)
		})
	})

	It("should handle array len", func() {
		runWithValue(cacheInstance, mockReader, 1, func(rd internal.Reader) (int, error) {
			return rd.ReadArrayLen()
		})
	})

	It("should handle fixed map len", func() {
		runWithoutValue(cacheInstance, mockReader, func(rd internal.Reader) error {
			return rd.ReadFixedMapLen(1)
		})
	})

	It("should handle map len", func() {
		runWithValue(cacheInstance, mockReader, 1, func(rd internal.Reader) (int, error) {
			return rd.ReadMapLen()
		})
	})

	It("should handle discard next", func() {
		runWithoutValue(cacheInstance, mockReader, func(rd internal.Reader) error {
			return rd.DiscardNext()
		})
	})

	It("should detect type mismatch between read and write", func() {
		readerFunc := func(rd internal.Reader) error {
			_, err := rd.ReadString()
			if err != nil {
				return err
			}
			return nil
		}

		spy, wrappedReader := cacheInstance.SpyReader(readerFunc)

		err := wrappedReader(mockReader)
		Expect(err).NotTo(HaveOccurred())

		key := "test_key"
		spy.StoreInCache(key)

		_, foundAfterWait := waitGet(cacheInstance, key)
		Expect(foundAfterWait).To(BeTrue())

		replayedReader, found, err := cacheInstance.GetKey(key)
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())

		value, err := replayedReader.ReadInt()
		Expect(value).To(Equal(int64(0)))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("type mismatch"))
	})

	It("should handle unserializable keys at set", func() {
		readerFunc := func(rd internal.Reader) error {
			_, err := rd.ReadString()
			if err != nil {
				return err
			}
			return nil
		}

		spy, wrappedReader := cacheInstance.SpyReader(readerFunc)

		err := wrappedReader(mockReader)
		Expect(err).NotTo(HaveOccurred())

		key := make(chan int)
		stored, err := spy.StoreInCache(key)
		Expect(stored).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})

	It("should handle unserializable keys at get", func() {
		key := make(chan int)
		replayedReader, found, err := cacheInstance.GetKey(key)
		Expect(replayedReader).To(BeNil())
		Expect(found).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})

	It("should handle unserializable keys at clear", func() {
		key := make(chan int)
		err := cacheInstance.ClearKey(key)
		Expect(err).To(HaveOccurred())
	})

	It("should be able to clear all", func() {
		readerFunc := func(rd internal.Reader) error {
			_, err := rd.ReadString()
			if err != nil {
				return err
			}
			return nil
		}

		spy, wrappedReader := cacheInstance.SpyReader(readerFunc)

		err := wrappedReader(mockReader)
		Expect(err).NotTo(HaveOccurred())

		key := "test_key"
		stored, err := spy.StoreInCache(key)
		Expect(stored).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())

		_, foundAfterWait := waitGet(cacheInstance, key)
		Expect(foundAfterWait).To(BeTrue())

		replayedReader, found, err := cacheInstance.GetKey(key)
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())

		value, err := replayedReader.ReadString()
		Expect(value).To(Equal("foo"))
		Expect(err).NotTo(HaveOccurred())

		cacheInstance.Clear()

		replayedReader, found, err = cacheInstance.GetKey(key)
		Expect(replayedReader).To(BeNil())
		Expect(found).To(BeFalse())
		Expect(err).NotTo(HaveOccurred())
	})
})

func runWithValue[T any](cache *cache.Cache, reader internal.Reader, expectedValue T, readerDriver func(internal.Reader) (T, error)) {
	readerFunc := func(rd internal.Reader) error {
		_, err := readerDriver(rd)
		if err != nil {
			return err
		}
		return nil
	}

	spy, wrappedReader := cache.SpyReader(readerFunc)

	err := wrappedReader(reader)
	Expect(err).NotTo(HaveOccurred())

	metrics := cache.Metrics()
	keysAdded := metrics.KeysAdded()
	Expect(keysAdded).To(Equal(uint64(0)))

	key := "test_key"
	stored, err := spy.StoreInCache(key)
	Expect(stored).To(BeTrue())
	Expect(err).NotTo(HaveOccurred())

	_, foundAfterWait := waitGet(cache, key)
	Expect(foundAfterWait).To(BeTrue())

	metrics = cache.Metrics()
	keysAdded = metrics.KeysAdded()
	Expect(keysAdded).To(Equal(uint64(1)))

	replayedReader, found, err := cache.GetKey(key)
	Expect(found).To(BeTrue())
	Expect(err).NotTo(HaveOccurred())

	value, err := readerDriver(replayedReader)
	Expect(err).NotTo(HaveOccurred())
	Expect(value).To(Equal(expectedValue))

	err = cache.ClearKey(key)
	Expect(err).NotTo(HaveOccurred())

	replayedReaderAfterClear, foundAfterClear, err := cache.GetKey(key)
	Expect(foundAfterClear).To(BeFalse())
	Expect(replayedReaderAfterClear).To(BeNil())
	Expect(err).NotTo(HaveOccurred())
}

func runWithoutValue(cache *cache.Cache, reader internal.Reader, readerDriver func(internal.Reader) error) {
	readerFunc := func(rd internal.Reader) error {
		return readerDriver(rd)
	}

	spy, wrappedReader := cache.SpyReader(readerFunc)

	err := wrappedReader(reader)
	Expect(err).NotTo(HaveOccurred())

	key := "test_key"
	spy.StoreInCache(key)

	_, foundAfterWait := waitGet(cache, key)
	Expect(foundAfterWait).To(BeTrue())

	replayedReader, found, err := cache.GetKey(key)
	Expect(found).To(BeTrue())
	Expect(err).NotTo(HaveOccurred())

	value, err := replayedReader.ReadReply()
	Expect(value).To(BeNil())
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("no more data"))

	err = readerDriver(replayedReader)
	Expect(err).NotTo(HaveOccurred())

	err = cache.ClearKey(key)
	Expect(err).NotTo(HaveOccurred())

	replayedReaderAfterClear, foundAfterClear, err := cache.GetKey(key)
	Expect(foundAfterClear).To(BeFalse())
	Expect(replayedReaderAfterClear).To(BeNil())
	Expect(err).NotTo(HaveOccurred())
}

type MockInternalReader struct {
	internal.Reader
}

func NewMockInternalReader() *MockInternalReader {
	return &MockInternalReader{}
}

func (m *MockInternalReader) Peek(n int) ([]byte, error) {
	return bytes.Repeat([]byte{'a'}, n), nil
}

func (m *MockInternalReader) PeekReplyType() (byte, error) {
	return 'x', nil
}

func (m *MockInternalReader) ReadLine() ([]byte, error) {
	return []byte("line"), nil
}

func (m *MockInternalReader) ReadReply() (interface{}, error) {
	return "reply", nil
}

func (m *MockInternalReader) ReadInt() (int64, error) {
	return 42, nil
}

func (m *MockInternalReader) ReadUint() (uint64, error) {
	return uint64(42), nil
}

func (m *MockInternalReader) ReadFloat() (float64, error) {
	return 3.14, nil
}

func (m *MockInternalReader) ReadString() (string, error) {
	return "foo", nil
}

func (m *MockInternalReader) ReadBool() (bool, error) {
	return true, nil
}

func (m *MockInternalReader) ReadSlice() ([]interface{}, error) {
	return []interface{}{"slice"}, nil
}

func (m *MockInternalReader) ReadFixedArrayLen(fixedLen int) error {
	if fixedLen != 1 {
		return errors.New("unexpected length")
	}
	return nil
}

func (m *MockInternalReader) ReadArrayLen() (int, error) {
	return 1, nil
}

func (m *MockInternalReader) ReadFixedMapLen(fixedLen int) error {
	if fixedLen != 1 {
		return errors.New("unexpected length")
	}
	return nil
}

func (m *MockInternalReader) ReadMapLen() (int, error) {
	return 1, nil
}

func (m *MockInternalReader) DiscardNext() error {
	return nil
}

// waitGet waits for a key to appear in the cache, compensating for the async
// nature of the ristretto library.
func waitGet(cache *cache.Cache, key interface{}) (internal.Reader, bool) {
	for i := 0; i < 10; i++ {
		val, found, err := cache.GetKey(key)
		if err != nil {
			return nil, false
		}
		//Expect(err).NotTo(HaveOccurred())
		if found {
			return val, found
		}
		time.Sleep(10 * time.Millisecond)
	}
	//Fail("Key not available in cache")
	return nil, false
}
