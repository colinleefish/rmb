package pgarray

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UUIDArray maps Go []uuid.UUID to Postgres uuid[] (GORM/pg driver default is wrong).
type UUIDArray []uuid.UUID

func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, u := range a {
		parts[i] = u.String()
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func (a *UUIDArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	var s string
	switch v := value.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("pgarray.UUIDArray: cannot scan %T", value)
	}
	return a.scanString(s)
}

func (a *UUIDArray) scanString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		*a = UUIDArray{}
		return nil
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return fmt.Errorf("pgarray.UUIDArray: invalid array literal %q", s)
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(inner, ",")
	out := make(UUIDArray, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		u, err := uuid.Parse(p)
		if err != nil {
			return fmt.Errorf("pgarray.UUIDArray: parse %q: %w", p, err)
		}
		out = append(out, u)
	}
	*a = out
	return nil
}
