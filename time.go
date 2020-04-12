package zerosvc

import (
	"strconv"
	"strings"
	"time"
)

// Time defines a flexible timestamp that is serialized as float but deserialized from int/float/text
type Time time.Time

// MarshalJSON is used to convert the timestamp to JSON
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

// UnmarshalJSON is used to convert the timestamp from JSON
func (t *Time) UnmarshalJSON(s []byte) (err error) {
	r := string(s)
	if strings.Contains(r, "-") {
		*(*time.Time)(t), err = time.Parse(`"`+time.RFC3339+`"`, r)
		return err
	} else if strings.Contains(r, ".") {
		f, err := strconv.ParseFloat(r, 64)
		if err != nil {
			return err
		}
		*(*time.Time)(t) = time.Unix(int64(f), int64((f-float64(int64(f)))*float64(time.Second)))
	} else {
		q, err := strconv.ParseInt(r, 10, 64)
		if err != nil {
			return err
		}
		*(*time.Time)(t) = time.Unix(q, 0)
	}
	return nil
}
