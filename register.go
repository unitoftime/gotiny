package gotiny

import (
	"fmt"
	"hash/crc32"
	"reflect"
	"strconv"
	"unsafe"
)

// //--------------------------------------------------------------------------------
// // nameType: string
// type nameType string
// var defaultNameType = nameType("")
// func newNameType(str string) nameType {
// 	return nameType(str)
// }
// func decNameType(d *Decoder, p unsafe.Pointer) {
// 	l, val := int(d.decUint32()), (*nameType)(p)
// 	*val = nameType(d.buf[d.index : d.index+l])
// 	d.index += l
// }
// func (e *Encoder) encNameType(nt nameType) {
// 	s := string(nt)
// 	e.encUint32(uint32(len(s))); e.buf = append(e.buf, s...)
// }

//--------------------------------------------------------------------------------
// nameType: crc32
var hashTable = crc32.MakeTable(crc32.IEEE)
func crc(label string) uint32 {
	return crc32.Checksum([]byte(label), hashTable)
}

type nameType uint32
var defaultNameType = nameType(0)
func newNameType(str string) nameType {
	return nameType(crc(str))
}
func decNameType(d *Decoder, p unsafe.Pointer) {
	*(*uint32)(p) = d.decUint32()
}
func (e *Encoder) encNameType(nt nameType) {
	e.encUint32(uint32(nt))
}


//--------------------------------------------------------------------------------

var (
	type2name = map[reflect.Type]nameType{}
	name2type = map[nameType]reflect.Type{}
)

func GetName(obj any) string {
	return GetNameByType(reflect.TypeOf(obj))
}
func GetNameByType(rt reflect.Type) string {
	return string(getName([]byte(nil), rt))
}

func getName(prefix []byte, rt reflect.Type) []byte {
	if rt == nil || rt.Kind() == reflect.Invalid {
		return append(prefix, []byte("<nil>")...)
	}
	if rt.Name() == "" { //structured type
		switch rt.Kind() {
		case reflect.Ptr:
			return getName(append(prefix, '*'), rt.Elem())
		case reflect.Array:
			return getName(append(prefix, "["+strconv.Itoa(rt.Len())+"]"...), rt.Elem())
		case reflect.Slice:
			return getName(append(prefix, '[', ']'), rt.Elem())
		case reflect.Struct:
			prefix = append(prefix, "struct {"...)
			nf := rt.NumField()
			if nf > 0 {
				prefix = append(prefix, ' ')
			}
			for i := 0; i < nf; i++ {
				field := rt.Field(i)
				if field.Anonymous {
					prefix = getName(prefix, field.Type)
				} else {
					prefix = getName(append(prefix, field.Name+" "...), field.Type)
				}
				if i != nf-1 {
					prefix = append(prefix, ';', ' ')
				} else {
					prefix = append(prefix, ' ')
				}
			}
			return append(prefix, '}')
		case reflect.Map:
			return getName(append(getName(append(prefix, "map["...), rt.Key()), ']'), rt.Elem())
		case reflect.Interface:
			prefix = append(prefix, "interface {"...)
			nm := rt.NumMethod()
			if nm > 0 {
				prefix = append(prefix, ' ')
			}
			for i := 0; i < nm; i++ {
				method := rt.Method(i)
				fn := getName([]byte(nil), method.Type)
				prefix = append(prefix, method.Name+string(fn[4:])...)
				if i != nm-1 {
					prefix = append(prefix, ';', ' ')
				} else {
					prefix = append(prefix, ' ')
				}
			}
			return append(prefix, '}')
		case reflect.Func:
			prefix = append(prefix, "func("...)
			for i := 0; i < rt.NumIn(); i++ {
				prefix = getName(prefix, rt.In(i))
				if i != rt.NumIn()-1 {
					prefix = append(prefix, ',', ' ')
				}
			}
			prefix = append(prefix, ')')
			no := rt.NumOut()
			if no > 0 {
				prefix = append(prefix, ' ')
			}
			if no > 1 {
				prefix = append(prefix, '(')
			}
			for i := 0; i < no; i++ {
				prefix = getName(prefix, rt.Out(i))
				if i != no-1 {
					prefix = append(prefix, ',', ' ')
				}
			}
			if no > 1 {
				prefix = append(prefix, ')')
			}
			return prefix
		}
	}

	if rt.PkgPath() == "" {
		prefix = append(prefix, rt.Name()...)
	} else {
		prefix = append(prefix, rt.PkgPath()+"."+rt.Name()...)
	}
	return prefix
}

func getNameOfType(rt reflect.Type) nameType {
	if name, has := type2name[rt]; has {
		return name
	}
	panic("gotiny: attempt to serialize unregistered type: " + GetNameByType(rt))
	// return registerType(rt)
}

func Register(i any) string {
	return registerType(reflect.TypeOf(i))
}

func registerType(rt reflect.Type) string {
	name := GetNameByType(rt)
	RegisterName(name, rt)
	return name
}

func RegisterName(name string, rt reflect.Type) {
	if name == "" {
		panic("attempt to register empty name")
	}

	if rt == nil || rt.Kind() == reflect.Invalid {
		panic("attempt to register nil type or invalid type")
	}

	if _, has := type2name[rt]; has {
		panic("gotiny: registering duplicate types for " + GetNameByType(rt))
	}

	nt := newNameType(name)
	if existingType, has := name2type[nt]; has {
		oldNameType := type2name[existingType]
		panic(fmt.Sprintf("gotiny: registered name collision: New: [%s (%s) (%d)] Old: [%s (%s) (%v)]",
			name, rt.String(), nt,
			"<Unknown>", existingType.String(), oldNameType,
		))
	}
	name2type[nt] = rt
	type2name[rt] = nt
}
