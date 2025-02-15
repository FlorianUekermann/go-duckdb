package duckdb

/*
#include <duckdb.h>
*/
import "C"

import (
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"time"
	"unsafe"
)

type rows struct {
	res    *C.duckdb_result
	s      *stmt
	cursor int64
}

func (r *rows) Columns() []string {
	if r.res == nil {
		panic("database/sql/driver: misuse of duckdb driver: Columns of closed rows")
	}

	colCount := C.duckdb_column_count(r.res)
	cols := make([]string, int64(colCount))
	for i := C.idx_t(0); i < colCount; i++ {
		name := C.duckdb_column_name(r.res, i)
		cols[i] = C.GoString(name)
	}

	return cols
}

func (r *rows) Next(dst []driver.Value) error {
	if r.res == nil {
		panic("database/sql/driver: misuse of duckdb driver: Next of closed rows")
	}

	rowCount := C.duckdb_row_count(r.res)
	if r.cursor >= int64(rowCount) {
		return io.EOF
	}

	colCount := C.duckdb_column_count(r.res)
	for i := 0; i < int(colCount); i++ {
		colType := C.duckdb_column_type(r.res, C.idx_t(i))
		colData := C.duckdb_column_data(r.res, C.idx_t(i))
		switch colType {
		case C.DUCKDB_TYPE_INVALID:
			return errInvalidType
		case C.DUCKDB_TYPE_BOOLEAN:
			dst[i] = (*[1 << 31]bool)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_TINYINT:
			dst[i] = (*[1 << 31]int8)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_SMALLINT:
			dst[i] = (*[1 << 31]int16)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_INTEGER:
			dst[i] = (*[1 << 31]int32)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_BIGINT:
			dst[i] = (*[1 << 31]int64)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_FLOAT:
			dst[i] = (*[1 << 31]float32)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_DOUBLE:
			dst[i] = (*[1 << 31]float64)(unsafe.Pointer(colData))[r.cursor]
		case C.DUCKDB_TYPE_DATE:
			val := (*[1 << 31]C.duckdb_date_struct)(unsafe.Pointer(colData))[r.cursor]
			dst[i] = time.Date(
				int(val.year),
				time.Month(val.month),
				int(val.day),
				0, 0, 0, 0,
				time.UTC,
			)
		case C.DUCKDB_TYPE_VARCHAR:
			dst[i] = C.GoString((*[1 << 31]*C.char)(unsafe.Pointer(colData))[r.cursor])
		case C.DUCKDB_TYPE_TIMESTAMP:
			// TODO: Implement when availabe in DuckDB
		}
	}

	r.cursor++

	return nil
}

// implements driver.RowsColumnTypeScanType
func (r *rows) ColumnTypeScanType(index int) reflect.Type {
	colType := C.duckdb_column_type(r.res, C.idx_t(index))
	switch colType {
	case C.DUCKDB_TYPE_BOOLEAN:
		return reflect.TypeOf(true)
	case C.DUCKDB_TYPE_TINYINT:
		return reflect.TypeOf(int8(0))
	case C.DUCKDB_TYPE_SMALLINT:
		return reflect.TypeOf(int16(0))
	case C.DUCKDB_TYPE_INTEGER:
		return reflect.TypeOf(int(0))
	case C.DUCKDB_TYPE_BIGINT:
		return reflect.TypeOf(int64(0))
	case C.DUCKDB_TYPE_FLOAT:
		return reflect.TypeOf(float32(0))
	case C.DUCKDB_TYPE_DOUBLE:
		return reflect.TypeOf(float64(0))
	case C.DUCKDB_TYPE_DATE, C.DUCKDB_TYPE_TIMESTAMP:
		return reflect.TypeOf(time.Time{})
	case C.DUCKDB_TYPE_VARCHAR:
		return reflect.TypeOf("")
	}
	return nil
}

// implements driver.RowsColumnTypeScanType
func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	colType := C.duckdb_column_type(r.res, C.idx_t(index))
	switch colType {
	case C.DUCKDB_TYPE_BOOLEAN:
		return "BOOLEAN"
	case C.DUCKDB_TYPE_TINYINT:
		return "TINYINT"
	case C.DUCKDB_TYPE_SMALLINT:
		return "SMALLINT"
	case C.DUCKDB_TYPE_INTEGER:
		return "INT"
	case C.DUCKDB_TYPE_BIGINT:
		return "BIGINT"
	case C.DUCKDB_TYPE_FLOAT:
		return "FLOAT"
	case C.DUCKDB_TYPE_DOUBLE:
		return "DOUBLE"
	case C.DUCKDB_TYPE_DATE:
		return "DATE"
	case C.DUCKDB_TYPE_VARCHAR:
		return "VARCHAR"
	case C.DUCKDB_TYPE_TIMESTAMP:
		return "TIMESTAMP"
	}
	return ""
}

func (r *rows) Close() error {
	if r.res == nil {
		panic("database/sql/driver: misuse of duckdb driver: Close of already closed rows")
	}

	C.duckdb_destroy_result(r.res)

	r.res = nil
	if r.s != nil {
		r.s.rows = false
		r.s = nil
	}

	return nil
}

var (
	errInvalidType = errors.New("invalid data type")
)
