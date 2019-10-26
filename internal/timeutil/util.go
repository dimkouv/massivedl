package timeutil

import (
	"fmt"
	"log"
	"time"
)

func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

//Convert a float64 to time.Duration
//@param quantity - the quantity of time to convert
//@param unit - the time unit of conversion ("ns", "us" (or "µs"), "ms", "s", "m", "h")
func FloatToDuration(tQuantity float64, tUnit string) time.Duration {
	stringTime := fmt.Sprintf("%g", tQuantity) + tUnit
	duration, err := time.ParseDuration(stringTime)

	if err != nil {
		log.Fatal(err)
	}

	return duration
}
