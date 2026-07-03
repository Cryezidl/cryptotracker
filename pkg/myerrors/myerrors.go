package myerrors

import "errors"

var (
	KeyNotFoundInCacheError = errors.New("key not found")
	ReadingCacheError       = errors.New("failed to read redis cache")
	FetchingCurrencyError   = errors.New("Could not fetch current currency data from available APIs")
)
