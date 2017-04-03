package mvc

import (
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

var (
	counter uint32
	randID  = make([]byte, 5)
	osPID   [2]byte
)

func init() {
	if _, err := rand.Read(randID); err != nil {
		panic("Unable to get random for SQLID generation")
	}
	pid := os.Getpid()
	osPID[0] = byte(pid >> 8)
	osPID[1] = byte(pid)
}

type SQLID string

func (id *SQLID) Scan(value interface{}) error {
	if s, ok := value.(string); ok {
		*id = SQLID(s)
	} else if b, ok := value.([]byte); ok {
		*id = SQLID(b)
	} else {
		*id = ""
	}
	return nil
}

func (id SQLID) Value() (driver.Value, error) {
	return string(id), nil
}

func (id SQLID) ToSQLNullID() SQLNullID {
	return SQLNullID{
		sql.NullString{
			String: string(id),
			Valid:  id != "",
		},
	}
}

type SQLNullString struct {
	sql.NullString
}

func (ns SQLNullString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.String)
	}
	return json.Marshal(nil)
}

type SQLNullID struct {
	sql.NullString
}

func (ns SQLNullID) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.String)
	}
	return json.Marshal(nil)
}

func NewSQLID() SQLID {
	var b [15]byte
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))

	b[4] = randID[0]
	b[5] = randID[1]
	b[6] = randID[2]
	b[7] = randID[3]
	b[8] = randID[4]

	b[9] = osPID[0]
	b[10] = osPID[1]

	binary.BigEndian.PutUint32(b[11:], atomic.AddUint32(&counter, 1))
	return SQLID(base32.StdEncoding.EncodeToString(b[:]))
}

func NewSQLNullID() SQLNullID {
	return SQLNullID{
		sql.NullString{
			String: "",
			Valid:  false,
		},
	}
}

func NewSQLNullString() SQLNullString {
	return SQLNullString{
		sql.NullString{
			String: "",
			Valid:  false,
		},
	}
}

func IsSQLID(id string) bool {
	return len(id) == 24
}

func SQLInsert(db *sql.DB, table string, m interface{}) (sql.Result, error) {
	if m == nil {
		return nil, nil
	}

	var v reflect.Value

	if v = reflect.ValueOf(m); v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Type().Kind() != reflect.Struct {
		return nil, nil
	}

	var (
		t            = v.Type()
		l            = t.NumField()
		fields       = make([]string, 0, l)
		values       = make([]interface{}, 0, l)
		valuesholder = make([]string, 0, l)
		query        *sql.Stmt
		err          error
	)

	for i := 0; i < l; i++ {
		if field := t.Field(i); field.PkgPath == "" {
			fname := field.Tag.Get("db")
			if fname == "" {
				fname = strings.ToLower(field.Name)
			}
			iface := v.Field(i).Interface()
			if t, ok := iface.(time.Time); ok && t.IsZero() {
				continue
			}
			fields = append(fields, fname)
			values = append(values, iface)
			valuesholder = append(valuesholder, "?")
		}
	}
	stmt := fmt.Sprintf("INSERT INTO `%s`(`%s`) VALUES(%s)",
		table,
		strings.Join(fields, "`,`"),
		strings.Join(valuesholder, ","))

	if query, err = db.Prepare(stmt); err == nil {
		return query.Exec(values...)
	}

	return nil, err
}
