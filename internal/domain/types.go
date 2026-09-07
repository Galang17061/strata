package domain

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var Unzoned = time.FixedZone("", 0)

type DateTime struct {
	time.Time
}

func Now() DateTime {
	return DateTime{time.Now()}
}

func UtcNow() DateTime {
	return DateTime{time.Now().UTC()}
}

func WallClock(t time.Time) DateTime {
	return DateTime{time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), Unzoned)}
}

func (d DateTime) Text() string {
	text := d.Time.Format("2006-01-02T15:04:05")
	if fraction := d.Time.Nanosecond() / 100; fraction != 0 {
		digits := strings.TrimRight(fmt.Sprintf("%07d", fraction), "0")
		text += "." + digits
	}
	switch d.Time.Location() {
	case time.UTC:
		text += "Z"
	case Unzoned:
	default:
		text += d.Time.Format("-07:00")
	}
	return text
}

func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Text() + `"`), nil
}

func (d *DateTime) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	if raw == "null" || raw == "" {
		return errors.New("The JSON value could not be converted to System.DateTime.")
	}
	parsed, err := ParseDateTime(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func ParseDateTime(raw string) (DateTime, error) {
	layouts := []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, raw, Unzoned)
		if err != nil {
			continue
		}
		if strings.HasSuffix(raw, "Z") {
			return DateTime{parsed.UTC()}, nil
		}
		if parsed.Location() != Unzoned {
			return DateTime{parsed.In(time.Local)}, nil
		}
		return DateTime{parsed}, nil
	}
	return DateTime{}, errors.New("The JSON value could not be converted to System.DateTime.")
}

func (d *DateTime) Scan(src any) error {
	switch value := src.(type) {
	case time.Time:
		*d = WallClock(value)
		return nil
	case nil:
		*d = DateTime{}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into DateTime", src)
	}
}

func (d DateTime) Value() (driver.Value, error) {
	return d.Time, nil
}

type Number struct {
	decimal.Decimal
}

func NewNumber(d decimal.Decimal) Number {
	return Number{d}
}

func NumberFromInt(value int64) Number {
	return Number{decimal.NewFromInt(value)}
}

func NumberFromString(text string) (Number, error) {
	d, err := decimal.NewFromString(text)
	return Number{d}, err
}

func (n Number) Text() string {
	if n.Decimal.Exponent() < 0 {
		return n.Decimal.StringFixed(-n.Decimal.Exponent())
	}
	return n.Decimal.String()
}

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(n.Text()), nil
}

func (n *Number) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if raw == "null" {
		return nil
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return errors.New("The JSON value could not be converted to System.Decimal.")
	}
	n.Decimal = d
	return nil
}

func (n Number) Value() (driver.Value, error) {
	return n.Text(), nil
}

func (n *Number) Scan(src any) error {
	return n.Decimal.Scan(src)
}

func NumberPtr(n Number) *Number {
	return &n
}

type Guid [16]byte

var EmptyGuid Guid

func ParseGuid(raw string) (Guid, error) {
	cleaned := strings.NewReplacer("-", "", "{", "", "}", "", "(", "", ")", "").Replace(strings.TrimSpace(raw))
	if len(cleaned) != 32 {
		return Guid{}, fmt.Errorf("The value '%s' is not valid.", raw)
	}
	bytes, err := hex.DecodeString(cleaned)
	if err != nil {
		return Guid{}, fmt.Errorf("The value '%s' is not valid.", raw)
	}
	var g Guid
	copy(g[:], bytes)
	return g, nil
}

func NewGuid() Guid {
	var g Guid
	_, _ = rand.Read(g[:])
	g[6] = (g[6] & 0x0f) | 0x40
	g[8] = (g[8] & 0x3f) | 0x80
	return g
}

func (g Guid) String() string {
	text := hex.EncodeToString(g[:])
	return text[0:8] + "-" + text[8:12] + "-" + text[12:16] + "-" + text[16:20] + "-" + text[20:]
}

func (g Guid) IsEmpty() bool {
	return g == EmptyGuid
}

func (g Guid) MarshalJSON() ([]byte, error) {
	return []byte(`"` + g.String() + `"`), nil
}

func (g *Guid) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if raw == "null" {
		return nil
	}
	parsed, err := ParseGuid(raw)
	if err != nil {
		return errors.New("The JSON value could not be converted to System.Guid.")
	}
	*g = parsed
	return nil
}

func (g *Guid) Scan(src any) error {
	switch value := src.(type) {
	case string:
		parsed, err := ParseGuid(value)
		if err != nil {
			return err
		}
		*g = parsed
		return nil
	case []byte:
		if len(value) == 16 {
			copy(g[:], value)
			return nil
		}
		parsed, err := ParseGuid(string(value))
		if err != nil {
			return err
		}
		*g = parsed
		return nil
	case nil:
		*g = EmptyGuid
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Guid", src)
	}
}

func (g Guid) Value() (driver.Value, error) {
	return g.String(), nil
}
