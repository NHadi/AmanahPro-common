package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const customDateFormat = "2006-01-02"

// CustomDate is a wrapper for time.Time to handle custom date formats
type CustomDate struct {
	time.Time
}

// UnmarshalJSON parses JSON string into CustomDate
func (cd *CustomDate) UnmarshalJSON(b []byte) error {
	// Remove quotes around JSON string
	str := string(b)
	if len(str) >= 2 {
		str = str[1 : len(str)-1]
	}

	parsedTime, err := time.Parse(customDateFormat, str)
	if err != nil {
		return fmt.Errorf("invalid date format, expected 'YYYY-MM-DD': %w", err)
	}

	cd.Time = parsedTime
	return nil
}

// MarshalJSON converts CustomDate back to JSON
func (cd CustomDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", cd.Time.Format(customDateFormat))), nil
}

// Value converts CustomDate to a value that can be stored in the database
func (cd CustomDate) Value() (driver.Value, error) {
	return cd.Time.Format(customDateFormat), nil
}

// Scan retrieves a CustomDate value from the database
func (cd *CustomDate) Scan(value interface{}) error {
	if value == nil {
		*cd = CustomDate{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		cd.Time = v
	case string:
		parsedTime, err := time.Parse(customDateFormat, v)
		if err != nil {
			return fmt.Errorf("invalid date format in database, expected 'YYYY-MM-DD': %w", err)
		}
		cd.Time = parsedTime
	default:
		return fmt.Errorf("unsupported type for CustomDate: %T", value)
	}
	return nil
}
