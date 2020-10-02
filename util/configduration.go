package util

import (
	"fmt"
	"strings"
	"time"
)

/*
	based on https://robreid.io/json-time-duration/

*/

type ConfigDuration struct {
	Duration time.Duration
}

func (d *ConfigDuration) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

func (d ConfigDuration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Duration.String())), nil
}

func (d *ConfigDuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return nil
	}
	d.Duration, err = time.ParseDuration(strings.Trim(buf, `"`))
	return err
}

func (d ConfigDuration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}
