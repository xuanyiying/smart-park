package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"
	"unsafe"
)

func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func SHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func Base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func RandomString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

func RandomInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

func StringsJoin(slice []string, sep string) string {
	return strings.Join(slice, sep)
}

func StringsSplit(s, sep string) []string {
	return strings.Split(s, sep)
}

func StringsTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func StringsToLower(s string) string {
	return strings.ToLower(s)
}

func StringsToUpper(s string) string {
	return strings.ToUpper(s)
}

func StringsContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func StringsHasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func StringsHasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func MapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func MapClone[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func Clone[T any](slice []T) []T {
	if slice == nil {
		return nil
	}
	result := make([]T, len(slice))
	copy(result, slice)
	return result
}

func Reverse[T any](slice []T) []T {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}

func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	chunks := make([][]T, 0, (len(slice)+size-1)/size)
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

func Now() time.Time {
	return time.Now()
}

func NowUnix() int64 {
	return time.Now().Unix()
}

func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}

func NowUnixNano() int64 {
	return time.Now().UnixNano()
}

func FormatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

func ParseTime(s, layout string) (time.Time, error) {
	return time.Parse(layout, s)
}

func MustParseTime(s, layout string) time.Time {
	t, err := time.Parse(layout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func DurationBetween(start, end time.Time) time.Duration {
	return end.Sub(start)
}

func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func UnixMilliToTime(milli int64) time.Time {
	return time.UnixMilli(milli)
}

func StringToPtr(s string) *string {
	return &s
}

func IntToPtr(i int) *int {
	return &i
}

func Int64ToPtr(i int64) *int64 {
	return &i
}

func BoolToPtr(b bool) *bool {
	return &b
}

func PtrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func PtrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func PtrToInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func PtrToBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func Min[T ~int | ~int64 | ~float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T ~int | ~int64 | ~float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Clamp[T ~int | ~int64 | ~float64](v, min, max T) T {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func InRange[T ~int | ~int64 | ~float64](v, min, max T) bool {
	return v >= min && v <= max
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func AbsInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func ZeroValue[T any]() T {
	var result T
	return result
}

func Fill[T any](slice []T, value T) {
	for i := range slice {
		slice[i] = value
	}
}

func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func Map[T any, R any](slice []T, transform func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

func Reduce[T any, R any](slice []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, item := range slice {
		result = reducer(result, item)
	}
	return result
}

func Any[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}
	return false
}

func All[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if !predicate(item) {
			return false
		}
	}
	return true
}

func None[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return false
		}
	}
	return true
}

func First[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[0], true
}

func Last[T any](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	return slice[len(slice)-1], true
}

func Flatten[T any](slices [][]T) []T {
	totalLen := 0
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func GroupBy[T any, K comparable](slice []T, keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range slice {
		key := keyFunc(item)
		result[key] = append(result[key], item)
	}
	return result
}

func Sleep(d time.Duration) {
	time.Sleep(d)
}

func Panic(v any) {
	panic(v)
}

func Recover(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	fn()
	return
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Try[T any](fn func() T, catch func(any)) T {
	defer func() {
		if r := recover(); r != nil && catch != nil {
			catch(r)
		}
	}()
	return fn()
}

func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func Copy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func NopCloser(r io.Reader) io.ReadCloser {
	return io.NopCloser(r)
}

func StringToBytes(s string) []byte {
	return []byte(s)
}

func BytesToString(b []byte) string {
	return string(b)
}

func StringToBytesNoAlloc(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func BytesToStringNoAlloc(b []byte) string {
	return unsafe.String(&b[0], len(b))
}
