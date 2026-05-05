package optional

import (
	"cmp"
	"fmt"
	"strconv"
)

// OptAllowed represents a constraint for the types allowed to be created as an optional value.
type OptAllowed interface {
	cmp.Ordered | bool
}

// Optional represents an optional value.
type Optional[T OptAllowed] struct {
	value     T
	isPresent bool
}

// New creates an optional value.
func New[T OptAllowed](v T) Optional[T] {
	return Optional[T]{v, true}
}

// Get returns the stored value.
func (o *Optional[T]) Get() (T, bool) {
	return o.value, o.isPresent
}

// Set sets the value.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.isPresent = true
}

// Value returns the value or an empty value if present.
func (o *Optional[T]) Value() T {
	return o.value
}

// IsZero is used to satisfy the 'omitzero' check for encoding/json.
func (o Optional[T]) IsZero() bool {
	return !o.isPresent
}

// MarshalJSON implements the json.Marshaler interface.
func (o Optional[T]) MarshalJSON() (data []byte, err error) {
	return o.MarshalText()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	return o.UnmarshalText(data)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (o Optional[T]) MarshalText() (data []byte, err error) {
	str := ""

	switch v := any(o.value).(type) {
	case bool:
		str = strconv.FormatBool(v)

	case int:
		str = strconv.FormatInt(int64(v), 10)

	case int8:
		str = strconv.FormatInt(int64(v), 10)

	case int16:
		str = strconv.FormatInt(int64(v), 10)

	case int32:
		str = strconv.FormatInt(int64(v), 10)

	case int64:
		str = strconv.FormatInt(int64(v), 10)

	case uint:
		str = strconv.FormatUint(uint64(v), 10)

	case uint8:
		str = strconv.FormatUint(uint64(v), 10)

	case uint16:
		str = strconv.FormatUint(uint64(v), 10)

	case uint32:
		str = strconv.FormatUint(uint64(v), 10)

	case uint64:
		str = strconv.FormatUint(uint64(v), 10)

	case float32:
		str = strconv.FormatFloat(float64(v), 'f', -1, 64)

	case float64:
		str = strconv.FormatFloat(float64(v), 'f', -1, 64)

	case string:
		str = v

	default:
		err = fmt.Errorf("marshalling optional value %v (%T) is not supported", v, v)
	}

	data = []byte(str)

	return
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (o *Optional[T]) UnmarshalText(data []byte) error {
	strdata := string(data)

	var val T
	var err error

	switch v := any(&val).(type) {
	case *bool:
		*v, err = strconv.ParseBool(strdata)

	case *int:
		vw, e := strconv.ParseInt(strdata, 10, 0)
		*v, err = int(vw), e

	case *int8:
		vw, e := strconv.ParseInt(strdata, 10, 8)
		*v, err = int8(vw), e

	case *int16:
		vw, e := strconv.ParseInt(strdata, 10, 16)
		*v, err = int16(vw), e

	case *int32:
		vw, e := strconv.ParseInt(strdata, 10, 32)
		*v, err = int32(vw), e

	case *int64:
		vw, e := strconv.ParseInt(strdata, 10, 64)
		*v, err = int64(vw), e

	case *uint:
		vw, e := strconv.ParseUint(strdata, 10, 0)
		*v, err = uint(vw), e

	case *uint8:
		vw, e := strconv.ParseUint(strdata, 10, 8)
		*v, err = uint8(vw), e

	case *uint16:
		vw, e := strconv.ParseUint(strdata, 10, 16)
		*v, err = uint16(vw), e

	case *uint32:
		vw, e := strconv.ParseUint(strdata, 10, 32)
		*v, err = uint32(vw), e

	case *uint64:
		vw, e := strconv.ParseUint(strdata, 10, 64)
		*v, err = uint64(vw), e

	case *float32:
		vw, e := strconv.ParseFloat(strdata, 32)
		*v, err = float32(vw), e

	case *float64:
		vw, e := strconv.ParseFloat(strdata, 64)
		*v, err = float64(vw), e

	case *string:
		*v, err = strdata, nil

	default:
		err = fmt.Errorf("unmarshalling optional value %v (%T) is not supported", v, v)
	}

	if err == nil {
		o.value = val
		o.isPresent = true
	}

	return err
}
