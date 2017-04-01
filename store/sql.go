package store

import (
	"database/sql"
	"errors"
)

type SQLScheme interface {
	Version() int
	Previous() SQLScheme
	CreateAdmin(user, pass, email string) (string, []interface{})
	Slice() []string
	UpdateVersion() string
}

func Migrate(s SQLScheme, db *sql.DB, v int, user, pass, email string) error {
	if s.Version() == v {
		return nil
	} else if s.Version() < v {
		return errors.New("SQL database scheme version is lower than current version")
	}

	if scheme := s.Previous(); scheme != nil {
		if err := Migrate(scheme, db, v, user, pass, email); err != nil {
			return err
		}
	}

	for _, stmt := range s.Slice() {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	if user != "" {
		if adm, i := s.CreateAdmin(user, pass, email); adm != "" {
			if _, err := db.Exec(adm, i...); err != nil {
				return err
			}
		}
	}

	if updatev := s.UpdateVersion(); updatev != "" {
		if _, err := db.Exec(updatev); err != nil {
			return err
		}
	}

	return nil
}
