package h

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestNow(t *testing.T) {
	now := Now()
	// Should be UTC
	assert.Equal(t, now.Location(), time.UTC)

	// Should be close to time.Now()
	diff := time.Since(now)
	assert.Equal(t, diff < time.Second, true)
}

func TestNowP(t *testing.T) {
	nowPtr := NowP()
	assert.NotEqual(t, nowPtr, nil)
	assert.Equal(t, nowPtr.Location(), time.UTC)
}

func TestNowPtrPlus(t *testing.T) {
	duration := time.Hour * 2
	futurePtr := NowPtrPlus(duration)

	assert.NotEqual(t, futurePtr, nil)
	assert.Equal(t, futurePtr.Location(), time.UTC)

	// Should be approximately 2 hours in the future
	diff := time.Until(*futurePtr)
	assert.Equal(t, diff > time.Hour, true)
	assert.Equal(t, diff < time.Hour*3, true)
}

func TestNowPlus(t *testing.T) {
	duration := time.Minute * 30
	future := NowPlus(duration)

	assert.Equal(t, future.Location(), time.UTC)

	// Should be approximately 30 minutes in the future
	diff := time.Until(future)
	assert.Equal(t, diff > time.Minute*29, true)
	assert.Equal(t, diff < time.Minute*31, true)
}

func TestDays(t *testing.T) {
	assert.Equal(t, Days(1), 24*time.Hour)
	assert.Equal(t, Days(7), 7*24*time.Hour)
	assert.Equal(t, Days(0), time.Duration(0))
	assert.Equal(t, Days(30), 30*24*time.Hour)
}

func TestTimeConstants(t *testing.T) {
	assert.Equal(t, OneMinute, time.Minute)
	assert.Equal(t, OneHour, time.Hour)
	assert.Equal(t, OneDay, 24*time.Hour)
	assert.Equal(t, OneWeek, 7*24*time.Hour)
	assert.Equal(t, OneMonth, 30*24*time.Hour)
	assert.Equal(t, OneYear, 365*24*time.Hour)
}
