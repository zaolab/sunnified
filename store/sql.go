package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type UserDetailGenerator func() (user, pass, email string)

type SQLScheme interface {
	Version() int
	Previous() SQLScheme
	CreateAdmin(UserDetailGenerator) (string, []interface{})
	Slice() []string
	UpdateVersion() string
}

func Migrate(s SQLScheme, db *sql.DB, v int, usrgen UserDetailGenerator) (err error) {
	if s.Version() == v {
		return
	} else if s.Version() < v {
		err = errors.New("SQL database scheme version is lower than current version")
		return
	}

	if scheme := s.Previous(); scheme != nil {
		if err = Migrate(scheme, db, v, usrgen); err != nil {
			return
		}
	}

	var tx *sql.Tx

	tx, err = db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if e := recover(); err != nil || e != nil {
			tx.Rollback()
			if e != nil {
				var ok bool
				if err, ok = e.(error); !ok {
					err = fmt.Errorf("%v", e)
				}
			}
		} else {
			err = tx.Commit()
		}

		return
	}()

	for _, stmt := range s.Slice() {
		if _, err = tx.Exec(stmt); err != nil {
			return
		}
	}

	if usrgen != nil {
		if adm, i := s.CreateAdmin(usrgen); adm != "" {
			if _, err = tx.Exec(adm, i...); err != nil {
				return
			}
		}
	}

	if updatev := s.UpdateVersion(); updatev != "" {
		if _, err = tx.Exec(updatev); err != nil {
			return
		}
	}

	return nil
}
