package response

import (
	"github.com/betterde/orbit/internal/pagination"
	"math"
	"net/http"
	"reflect"
)

type (
	Data struct {
		Meta  *pagination.Paginator `json:"meta,omitempty"`
		Item  interface{}           `json:"item"`
		Items interface{}           `json:"items"`
	}

	Response struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}
)

// Success 发送成功响应
func Success(message string, data interface{}, meta *pagination.Paginator) Response {
	if meta != nil {
		meta.Last = int64(math.Ceil(float64(meta.Total) / float64(meta.Limit)))
	}

	if data == nil {
		return Response{
			Code:    http.StatusOK,
			Message: message,
			Data: &Data{
				Meta:  meta,
				Item:  struct{}{},
				Items: []interface{}{},
			},
		}
	}

	v := reflect.TypeOf(data)
	switch v.Kind() {
	case reflect.Array:
	case reflect.Slice:
		return Response{
			Code:    http.StatusOK,
			Message: message,
			Data: &Data{
				Meta:  meta,
				Item:  struct{}{},
				Items: data,
			},
		}
	}

	return Response{
		Code:    http.StatusOK,
		Message: message,
		Data: &Data{
			Meta:  meta,
			Item:  data,
			Items: []interface{}{},
		},
	}
}

// UnAuthenticated 认证失败响应
func UnAuthenticated(message string) Response {
	return Response{
		Code:    http.StatusUnauthorized,
		Message: message,
		Data: &Data{
			Meta:  nil,
			Item:  struct{}{},
			Items: []interface{}{},
		},
	}
}

func NotFound(message string) Response {
	return Response{
		Code:    http.StatusNotFound,
		Message: message,
		Data: &Data{
			Meta:  nil,
			Item:  struct{}{},
			Items: []interface{}{},
		},
	}
}

func ValidationError(message string, err error) Response {
	return Response{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
		Data: &Data{
			Meta:  nil,
			Item:  err.Error(),
			Items: []interface{}{},
		},
	}
}

func InternalServerError(message string, err error) Response {
	return Response{
		Code:    http.StatusInternalServerError,
		Message: message,
		Data: &Data{
			Meta:  nil,
			Item:  err.Error(),
			Items: []interface{}{},
		},
	}
}

func Send(code int, message string, data interface{}) Response {
	v := reflect.TypeOf(data)
	switch v.Kind() {
	case reflect.Array:
	case reflect.Slice:
		return Response{
			Code:    code,
			Message: message,
			Data: &Data{
				Meta:  nil,
				Item:  struct{}{},
				Items: data,
			},
		}
	}

	return Response{
		Code:    code,
		Message: message,
		Data: &Data{
			Meta:  nil,
			Item:  data,
			Items: []interface{}{},
		},
	}
}
