package merge

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/goccy/go-json"
)

var (
	ErrDataJSON    = errors.New("merge: error in data JSON")
	ErrPatchJSON   = errors.New("merge: error in patch JSON")
	ErrMergedJSON  = errors.New("merge: error writing merged JSON")
	ErrPatchObject = errors.New("merge: patch value must be object")
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}

var readerPool = sync.Pool{
	New: func() any {
		return bytes.NewReader(nil)
	},
}

func getReader(data []byte) *bytes.Reader {
	r := readerPool.Get().(*bytes.Reader)
	r.Reset(data)
	return r
}

func putReader(r *bytes.Reader) {
	readerPool.Put(r)
}

type Info struct {
	Errors   []error
	Replaced map[string]any
}

func (info *Info) mergeValue(path []string, patch map[string]any, key string, value any, newKey bool) any {
	patchValue, patchHasValue := patch[key]

	if !patchHasValue {
		return value
	}

	_, patchValueIsObject := patchValue.(map[string]any)

	path = append(path, key)
	pathStr := strings.Join(path, ".")

	_, ok := value.(map[string]any)
	if ok {
		if !patchValueIsObject {
			err := fmt.Errorf("%w for key \"%v\"", ErrPatchObject, pathStr)
			info.Errors = append(info.Errors, err)
			return value
		}

		return info.mergeObjects(value, patchValue, path)
	}

	_, ok = value.([]any)
	if ok && patchValueIsObject {
		return info.mergeObjects(value, patchValue, path)
	}

	if !jsonValuesEqual(value, patchValue) || newKey {
		info.Replaced[pathStr] = patchValue
	}

	return patchValue
}

func (info *Info) mergeObjects(data, patch any, path []string) any {
	patchObject, ok := patch.(map[string]any)
	if ok {
		dataArray, ok := data.([]any)
		if ok {
			ret := make([]any, len(dataArray))

			for i, val := range dataArray {
				ret[i] = info.mergeValue(path, patchObject, strconv.Itoa(i), val, false)
			}

			return ret
		}

		dataObject, ok := data.(map[string]any)
		if ok {
			ret := make(map[string]any)

			founds := []string{}
			for k, v := range dataObject {
				ret[k] = info.mergeValue(path, patchObject, k, v, false)
				founds = append(founds, k)
			}

			for k, v := range patchObject {
				if !slices.Contains(founds, k) {
					ret[k] = info.mergeValue(path, patchObject, k, v, true)
				}
			}

			return ret
		}
	}

	return data
}

func jsonValuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case bool:
		vb, ok := b.(bool)
		return ok && va == vb
	case float64:
		vb, ok := b.(float64)
		return ok && va == vb
	case json.Number:
		vb, ok := b.(json.Number)
		return ok && va == vb
	default:
		ja, errA := json.Marshal(a)
		jb, errB := json.Marshal(b)
		if errA != nil || errB != nil {
			return false
		}
		return bytes.Equal(ja, jb)
	}
}

func Merge(data, patch any) (any, *Info) {
	info := &Info{
		Replaced: make(map[string]any),
	}
	ret := info.mergeObjects(data, patch, nil)
	return ret, info
}

func MergeBytes(dataBuff, patchBuff []byte) (mergedBuff []byte, info *Info, err error) {
	var data, patch, merged any

	err = unmarshalJSON(dataBuff, &data)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrDataJSON, err)
		return
	}

	err = unmarshalJSON(patchBuff, &patch)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrPatchJSON, err)
		return
	}

	merged, info = Merge(data, patch)

	buf := getBuffer()
	encoder := json.NewEncoder(buf)
	err = encoder.Encode(merged)
	if err != nil {
		putBuffer(buf)
		err = fmt.Errorf("%w: %w", ErrMergedJSON, err)
		return
	}

	bufData := buf.Bytes()
	if len(bufData) > 0 && bufData[len(bufData)-1] == '\n' {
		bufData = bufData[:len(bufData)-1]
	}
	mergedBuff = make([]byte, len(bufData))
	copy(mergedBuff, bufData)
	putBuffer(buf)

	return
}

func unmarshalJSON(buff []byte, data any) error {
	reader := getReader(buff)
	defer putReader(reader)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	return decoder.Decode(data)
}
