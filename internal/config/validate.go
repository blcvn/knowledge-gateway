package config

import (
	"errors"
	"fmt"
)

func (c Config) Validate() error {
	var errs []error

	if c.HTTP.Host == "" {
		errs = append(errs, errors.New("http host must not be empty"))
	}
	if c.HTTP.Port <= 0 {
		errs = append(errs, fmt.Errorf("http port must be positive: %d", c.HTTP.Port))
	}

	if c.Postgres.Host == "" {
		errs = append(errs, errors.New("postgres host must not be empty"))
	}
	if c.Postgres.Port <= 0 {
		errs = append(errs, fmt.Errorf("postgres port must be positive: %d", c.Postgres.Port))
	}
	if c.Postgres.User == "" {
		errs = append(errs, errors.New("postgres user must not be empty"))
	}
	if c.Postgres.Database == "" {
		errs = append(errs, errors.New("postgres database must not be empty"))
	}

	if c.Redis.Host == "" {
		errs = append(errs, errors.New("redis host must not be empty"))
	}
	if c.Redis.Port <= 0 {
		errs = append(errs, fmt.Errorf("redis port must be positive: %d", c.Redis.Port))
	}

	return errors.Join(errs...)
}
