package micro

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"
	"time"
	"unsafe"
)

var generator *ShortIDGenerator

// ShortIDGenerator creates short, URL-friendly IDs with high uniqueness guarantees
type ShortIDGenerator struct {
	alphabet   string
	lastTime   int64
	counter    uint32 // Changed to uint32 for atomic operations
	machineID  uint16
	timeOffset int64
	buffer     []byte
}

// NewShortIDGenerator creates a new generator with a machine ID
// machineID should be unique per server/instance (0-1023)
func NewShortIDGenerator(machineID int) *ShortIDGenerator {
	// Ensure machineID is in valid range (10 bits = 0-1023)
	machineID = machineID & 0x3FF

	// Calculate time offset once during initialization
	timeOffset := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	return &ShortIDGenerator{
		alphabet:   "0123456789abcdefghijklmnopqrstuvwxyz",
		lastTime:   0,
		counter:    0,
		machineID:  uint16(machineID),
		timeOffset: timeOffset,
		buffer:     make([]byte, 8),
	}
}

// byteSliceToString converts a byte slice to a string without allocation
func byteSliceToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Generate creates a new 8-character ID
func (g *ShortIDGenerator) Generate() string {
	// Get current time
	now := time.Now().UnixMilli()
	last := atomic.LoadInt64(&g.lastTime)

	var count uint32

	// Fast path: different millisecond
	if now != last {
		if atomic.CompareAndSwapInt64(&g.lastTime, last, now) {
			// Successfully updated timestamp, reset counter
			atomic.StoreUint32(&g.counter, 0)
			count = 0
		} else {
			// Another thread updated the timestamp; retry with updated values
			return g.Generate()
		}
	} else {
		// We're in the same millisecond, increment counter
		count = atomic.AddUint32(&g.counter, 1)
		// Check for overflow (only use lower 12 bits)
		if count&0xFFF >= 4096 {
			// Sleep until next millisecond
			time.Sleep(time.Millisecond)
			return g.Generate()
		}
	}

	// Only use lower 12 bits of counter
	count = count & 0xFFF

	// Calculate time delta from base time
	timestamp := now - g.timeOffset

	// Pack values (same bit allocation as original)
	combined := (uint64(timestamp) << 22) | (uint64(g.machineID) << 12) | uint64(count)

	// Convert to base36 using pre-allocated buffer
	for i := 7; i >= 0; i-- {
		g.buffer[i] = g.alphabet[combined%36]
		combined /= 36
	}

	return byteSliceToString(g.buffer)
}

// NewId generates a new ID with optional prefix
func NewId(prefix string) string {
	value := generator.Generate()
	if len(prefix) == 0 {
		return value
	}

	lastChar := prefix[len(prefix)-1]
	if lastChar == '_' || lastChar == '-' {
		return prefix + value
	}

	return prefix + "_" + value
}

// NewIdPtr returns a pointer to a new ID
func NewIdPtr(prefix string) *string {
	value := NewId(prefix)
	return &value
}

func InitIdGenerator(machineID int) {
	generator = NewShortIDGenerator(machineID)
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func NewRandomString(length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range result {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}
