package dates

import "time"

const OneMinute = time.Minute
const OneHour = time.Hour
const OneDay = 24 * OneHour
const OneWeek = 7 * OneDay
const OneMonth = 30 * OneDay
const OneYear = 365 * OneDay

func Now() time.Time {
	return time.Now().UTC()
}

func NowP() *time.Time {
	value := time.Now().UTC()
	return &value
}

func NowPtrPlus(d time.Duration) *time.Time {
	value := time.Now().UTC()
	value = value.Add(d)
	return &value
}

func NowPlus(d time.Duration) time.Time {
	value := time.Now().UTC()
	value = value.Add(d)
	return value
}

func Days(value int) time.Duration {
	return time.Hour * 24 * time.Duration(value)
}
