package swe

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	jsoniter "github.com/json-iterator/go"
)

type sweError struct {
	err  error
	code int
}

func (e sweError) Err() error { return e.err }
func (e sweError) Code() int  { return e.code }

type SweError interface {
	Err() error
	Code() int
}

func Error(code int, err error) SweError {
	return sweError{code: code, err: err}
}

func MakeAPIHandler[InType, OutType any](handler func(*Context, *InType) (*OutType, SweError)) HandlerFunc {
	return func(ctx *Context) {
		r := ctx.Request
		w := ctx.Response
		var param InType
		json := jsoniter.ConfigCompatibleWithStandardLibrary
		if r.Method == http.MethodGet {
			if err := DecodeForm(r, &param); err != nil {
				handleError(w, r, err)
				return
			}
		} else if r.Method == http.MethodPost {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				handleError(w, r, err)
				return
			}
			err = json.Unmarshal(data, &param)
			if err != nil {
				handleError(w, r, err)
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		outCode := 0
		outMsg := ""
		out, err := handler(ctx, &param)
		if err.Err() != nil {
			outCode = err.Code()
			outMsg = err.Err().Error()
			CtxLogger(ctx).Error("request %s failed: %v", r.URL.Path, err)
		}
		outMap := map[string]any{
			"code": outCode,
			"msg":  outMsg,
			"data": out,
		}

		outData, err2 := json.Marshal(outMap)
		if err2 != nil {
			handleError(w, r, err2)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(outData)
	}
}

func DecodeForm(r *http.Request, ptr any) error {
	rValue := reflect.ValueOf(ptr).Elem()
	rType := rValue.Type()

	for i := 0; i < rValue.NumField(); i++ {
		if !rValue.Field(i).CanSet() {
			continue
		}
		if tag := rType.Field(i).Tag.Get("form"); tag != "" {
			formValue := r.FormValue(tag)
			if formValue == "" {
				continue
			}

			switch rType.Field(i).Type.Kind() {
			case reflect.String:
				rValue.Field(i).SetString(formValue)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				value, err := strconv.ParseInt(formValue, 10, 64)
				if err != nil {
					return err
				}
				rValue.Field(i).SetInt(value)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				value, err := strconv.ParseUint(formValue, 10, 64)
				if err != nil {
					return err
				}
				rValue.Field(i).SetUint(value)
			default:
				return fmt.Errorf("invalid field type for %s: %T", tag, rValue.Field(i).Interface())
			}
		}
	}
	return nil
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprint(err)))
}
