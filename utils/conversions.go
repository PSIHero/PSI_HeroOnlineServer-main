package utils

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/guregu/null.v3"
)

func check(e error) {
	if err := recover(); err != nil {
		//log.Fatal(e)
	}
}

func abs(a time.Duration) time.Duration {
	if a >= 0 {
		return a
	}
	return -a
}
func DurationToString(d time.Duration) string {
	d = abs(d.Round(time.Minute))
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02dh %02dmin", h, m)
}

func ByteArrayToHexString(array []byte, n int) (hex string) {
	var buffer bytes.Buffer

	for i := 0; i < n; i++ {
		var (
			h string
		)
		if i < len(array) {
			h = fmt.Sprintf("%02X", array[i])
		} else {
			h = "00"
		}
		buffer.WriteString(h + " ")
	}
	hex = buffer.String()
	hex = hex[0 : len(hex)-1]
	return
}

func BytesToInt(arr []byte, reverse bool) int64 {

	var total int64
	_arr := make([]byte, len(arr))
	copy(_arr, arr)

	if reverse {
		_arr = reverseBytes(_arr)
	}

	for i := 0; i < len(_arr); i++ {
		power := int64(math.Pow(256, float64(len(_arr)-(i+1))))
		total += int64(_arr[i]) * power
	}

	return total
}

func IntToBytes(dec uint64, bytes byte, reverse bool) []byte {

	arr := make([]byte, 8)
	if dec == 0 {
		return arr[:bytes]
	}

	if bytes < 8 {
		dec = dec % (1 << (bytes * 8))
	}

	for i := 7; i >= 0; i-- {
		power := uint64(math.Pow(256, float64(i)))
		arr[i] = byte(dec / power)
		if arr[i] > 0 {
			dec -= uint64(arr[i]) * power
		}
	}

	if reverse {
		return arr[:bytes]
	}

	return reverseBytes(arr)[8-bytes : 8]
}

func FloatToBytes(dec float64, bytes byte, reverse bool) []byte {

	return IntToBytes(uint64(math.Float32bits(float32(dec))), bytes, reverse)
}

func BytesToFloat(arr []byte, reverse bool) float64 {
	return float64(math.Float32frombits(uint32(BytesToInt(arr, reverse))))
}

func reverseBytes(input []byte) []byte {
	if len(input) == 0 {
		return input
	}

	return append(reverseBytes(input[1:]), input[0])
}

func IntsToBytes(ints ...int) []byte {

	var arr []byte
	for _, i := range ints {
		arr = append(arr, byte(i))
	}
	return arr
}

func StringToInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Println(err, s)
		return 0
	}
	return i
}
func StringToFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Println(err)
		return 0
	}
	return f
}
func StringToBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		log.Println(err)
		return false
	}
	return b
}
func ParseDate(date null.Time) string {
	if date.Valid {
		year, month, day := date.Time.Date()
		return fmt.Sprintf("%02d.%02d.%d", day, month, year)
	}

	return ""
}
func RemoveAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, e := transform.String(t, s)
	if e != nil {
		panic(e)
	}
	return output
}
