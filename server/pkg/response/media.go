package response

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"
)

// 响应层统一签名媒体 URL。
//
// 媒体引用散落在数十个 DTO / map 里，逐个 service 改既容易漏，也会把
// OSS 依赖扩散到所有业务层。这里在 JSON 序列化前遍历一次 data，把
// file_url / thumb_url / cover_url / avatar / media_url / face_url
// 统一换成短期签名 URL；非 OSS 来源（微信头像等）原样保留。
const maxMediaSignDepth = 12

var (
	mediaSignerMu sync.RWMutex
	mediaSigner   func(ref string, thumb bool) string
)

// SetMediaSigner 在应用装配阶段注入签名函数；传 nil 可关闭响应层签名。
func SetMediaSigner(fn func(ref string, thumb bool) string) {
	mediaSignerMu.Lock()
	mediaSigner = fn
	mediaSignerMu.Unlock()
}

func signMedia(data any) any {
	mediaSignerMu.RLock()
	signer := mediaSigner
	mediaSignerMu.RUnlock()
	if signer == nil || data == nil {
		return data
	}
	signed := signValue(reflect.ValueOf(data), signer, 0)
	if !signed.IsValid() {
		return data
	}
	return signed.Interface()
}

var (
	rawMessageType = reflect.TypeOf(json.RawMessage(nil))
	timeType       = reflect.TypeOf(time.Time{})
)

func signValue(v reflect.Value, signer func(string, bool) string, depth int) reflect.Value {
	if !v.IsValid() || depth > maxMediaSignDepth {
		return v
	}

	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		signed := signValue(v.Elem(), signer, depth+1)
		if !signed.IsValid() || !signed.Type().AssignableTo(v.Type()) {
			return v
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(signed)
		return out

	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		elem := signValue(v.Elem(), signer, depth+1)
		if !elem.IsValid() || elem.Type() != v.Elem().Type() {
			return v
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(elem)
		return out

	case reflect.Struct:
		if v.Type() == timeType || v.Type() == rawMessageType {
			return v
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" { // unexported
				continue
			}
			key := jsonFieldName(field)
			if key == "" {
				continue
			}
			fv := v.Field(i)
			if isMediaKey(key) {
				if signed, ok := signedMediaValue(fv, signer, isThumbKey(key)); ok {
					out.Field(i).Set(signed)
				}
				continue
			}
			sv := signValue(fv, signer, depth+1)
			if sv.IsValid() && sv.Type() == fv.Type() && out.Field(i).CanSet() {
				out.Field(i).Set(sv)
			}
		}
		return out

	case reflect.Slice:
		if v.IsNil() || v.Type().Elem().Kind() == reflect.Uint8 {
			return v
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(signValue(v.Index(i), signer, depth+1))
		}
		return out

	case reflect.Array:
		out := reflect.New(v.Type()).Elem()
		out.Set(v)
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(signValue(v.Index(i), signer, depth+1))
		}
		return out

	case reflect.Map:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			val := iter.Value()
			if k.Kind() == reflect.String && isMediaKey(k.String()) {
				if signed, ok := signedMediaValue(val, signer, isThumbKey(k.String())); ok {
					out.SetMapIndex(k, signed)
					continue
				}
			}
			out.SetMapIndex(k, signValue(val, signer, depth+1))
		}
		return out

	default:
		return v
	}
}

func signedMediaValue(v reflect.Value, signer func(string, bool) string, thumb bool) (reflect.Value, bool) {
	switch v.Kind() {
	case reflect.String:
		return reflect.ValueOf(signer(v.String(), thumb)).Convert(v.Type()), true
	case reflect.Pointer:
		if v.IsNil() || v.Type().Elem().Kind() != reflect.String {
			return v, false
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().SetString(signer(v.Elem().String(), thumb))
		return out, true
	case reflect.Interface:
		if v.IsNil() {
			return v, false
		}
		elem := v.Elem()
		if elem.Kind() != reflect.String {
			return v, false
		}
		signed := reflect.ValueOf(signer(elem.String(), thumb))
		if !signed.Type().AssignableTo(v.Type()) {
			return v, false
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(signed)
		return out, true
	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.String {
			return v, false
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).SetString(signer(v.Index(i).String(), thumb))
		}
		return out, true
	default:
		return v, false
	}
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if tag == "" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}
	return name
}

func isMediaKey(key string) bool {
	switch strings.ToLower(key) {
	case "file_url", "thumb_url", "cover_url", "avatar", "media_url", "face_url",
		"fileurl", "thumburl", "coverurl", "mediaurl", "faceurl":
		return true
	default:
		return false
	}
}

func isThumbKey(key string) bool {
	k := strings.ToLower(key)
	return k == "thumb_url" || k == "thumburl"
}
