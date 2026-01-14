package mathutil

import (
	"errors"
	"fmt"
	"math"
)

var ErrOverflow = errors.New("value exceeds target type capacity")

func Uint64ToInt64(v uint64) (int64, error) {
	if v > math.MaxInt64 {
		return 0, fmt.Errorf("value %d overflows int64: %w", v, ErrOverflow)
	}
	return int64(v), nil
}

func Uint64ToInt(v uint64) (int, error) {
	if v > math.MaxInt {
		return 0, fmt.Errorf("value %d overflows int: %w", v, ErrOverflow)
	}
	return int(v), nil
}

func IntToUint64(v int) (uint64, error) {
	if v < 0 {
		return 0, fmt.Errorf("negative value %d cannot convert to uint64: %w", v, ErrOverflow)
	}
	return uint64(v), nil
}
