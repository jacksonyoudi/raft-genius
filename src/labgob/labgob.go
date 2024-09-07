package labgob

import (
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

var mu sync.Mutex
var errorCount int // for test capital
var checked map[reflect.Type]bool

type LabEncoder struct {
	gob *gob.Encoder
}

func NewEncoder(w io.Writer) *LabEncoder {
	enc := &LabEncoder{
		gob: gob.NewEncoder(w),
	}
	return enc
}

func (enc *LabEncoder) Encode(v interface{}) error {
	checkValue(v)
	return enc.gob.Encode(v)

}

func (enc *LabEncoder) EncodeValue(value reflect.Value) error {
	checkValue(value.Interface())
	return enc.gob.EncodeValue(value)
}

type LabDecoder struct {
	gob *gob.Decoder
}

func NewDecoder(r io.Reader) *LabDecoder {
	dec := &LabDecoder{
		gob: gob.NewDecoder(r),
	}
	return dec
}

func (d *LabDecoder) Decode(v interface{}) error {
	checkValue(v)
	checkDefault(v)
	return d.gob.Decode(v)
}

func Register(value interface{}) {
	checkValue(value)
	gob.Register(value)
}

func checkValue(v interface{}) {
	checkType(reflect.TypeOf(v))
}

func RegisterName(name string, value interface{}) {
	checkValue(value)
	gob.RegisterName(name, value)
}

func checkType(t reflect.Type) {
	k := t.Kind()       // 获取类型t的Kind
	mu.Lock()           // 加锁
	if checked == nil { // 如果checked为空
		checked = make(map[reflect.Type]bool) // 创建一个map，用于存储已经检查过的类型
	}
	if checked[t] { // 如果类型t已经检查过
		mu.Unlock() // 解锁
		return      // 返回
	}
	checked[t] = true // 将类型t标记为已经检查过
	mu.Unlock()       // 在函数返回前解锁

	switch k { // 根据类型t的Kind进行不同的处理
	case reflect.Struct: // 如果类型t是结构体
		for i := 0; i < t.NumField(); i++ { // 遍历结构体的所有字段
			f := t.Field(i)                            // 获取字段f
			rune, _ := utf8.DecodeRuneInString(f.Name) // 将字段名转换为rune
			if unicode.IsUpper(rune) == false {        // 如果字段名不是大写字母
				fmt.Printf("labgob error: lower-case field %v of %v in RPC or persist/snapshot will break your Raft\n",
					f.Name, t.Name()) // 输出错误信息
				mu.Lock()    // 加锁
				errorCount++ // 错误计数加1
				mu.Unlock()  // 解锁
			}
			checkType(f.Type) // 递归检查字段类型
		}
		return // 返回
	case reflect.Slice, reflect.Array, reflect.Ptr: // 如果类型t是切片、数组或指针
		checkType(t.Elem()) // 递归检查元素类型
		return              // 返回
	case reflect.Map: // 如果类型t是映射
		checkType(t.Elem()) // 递归检查元素类型
		checkType(t.Key())  // 递归检查键类型
		return              // 返回
	default: // 如果类型t不是上述类型
		return // 返回
	}

}

func checkDefault(value interface{}) {
	if value == nil {
		return
	}
	checkDefaultinner(reflect.ValueOf(value), 1, "")
}

func checkDefaultinner(value reflect.Value, depth int, name string) {
	if depth > 3 {
		return
	}
	t := value.Type()
	k := t.Kind()

	switch k {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			vv := value.Field(i)
			n := t.Field(i).Name
			if n != "" {
				name = name + "." + n
			}
			checkDefaultinner(vv, depth+1, name)
		}
		return

	case reflect.Ptr:
		if value.IsNil() {
			return
		}
		checkDefaultinner(value.Elem(), depth+1, name)

	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.String:
		// 如果反射得到的零值与value的值不相等
		if reflect.DeepEqual(reflect.Zero(t).Interface(), value.Interface()) == false {
			// 加锁
			mu.Lock()
			// 如果错误计数小于1
			if errorCount < 1 {
				// 定义变量what，用于存储变量或字段的名称
				what := name
				// 如果变量名为空
				if what == "" {
					// 将变量名设置为类型的名称
					what = t.Name()
				}
				// 打印警告信息
				fmt.Printf("labgob warning: Decoding into a non-default variable/field %v may not work\n",
					what)
			}
			// 错误计数加1
			errorCount++
			// 解锁
			mu.Unlock()
		}
		return
	}
}
