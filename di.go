package di

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	diTag   = "di"
	diType  = "type"
	diAlias = "alias"
)

var (
	di = New()
)

func Global() *DI {
	return di
}

func Register[T any](f func() T) {
	RegisterAlias[T]("", f)
}

func RegisterAlias[T any](name string, f func() T) {
	RegisterAliasDI[T](di, name, f)
}

func Get[T any]() T {
	return GetAlias[T]("")
}

func GetAlias[T any](name string) T {
	return GetDI[T](di)
}

type alias struct {
	t    reflect.Type
	name string
}

type Cleaner interface {
	Clean()
}

type DI struct {
	typeBuilder map[alias]func() any
	typeRepo    map[alias]reflect.Value
}

func New() *DI {
	di := &DI{
		typeBuilder: make(map[alias]func() any),
		typeRepo:    make(map[alias]reflect.Value),
	}
	return di
}

func RegisterDI[T any](d *DI, f func() T) {
	RegisterAliasDI(d, "", f)
}

func RegisterAliasDI[T any](d *DI, name string, f func() T) {
	rt := reflect.TypeFor[T]()
	key := alias{
		t:    rt,
		name: name,
	}
	_, ok := d.typeBuilder[key]
	assert(!ok, fmt.Errorf("duplicate register, type: %s, name: %s", rt.Name(), name))
	d.typeBuilder[key] = func() any {
		return f()
	}
}

func GetDI[T any](d *DI) T {
	return GetAliasDI[T](d, "")
}

func GetAliasDI[T any](d *DI, name string) T {
	rt := reflect.TypeFor[T]()
	var instance T
	reflect.ValueOf(&instance).Elem().Set(build(d, alias{t: rt, name: name}))
	return instance
}

func build(d *DI, a alias) reflect.Value {
	v, ok := d.typeRepo[a]
	if ok {
		return v
	}
	f, ok := d.typeBuilder[a]
	assert(ok, fmt.Errorf("builder absence, type: %s, name: %s", a.t.Name(), a.name))
	value := reflect.ValueOf(f())
	d.typeRepo[a] = value
	buildStruct(d, value)
	return value
}

func buildStruct(d *DI, v reflect.Value) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get(diTag)
		if tag == "" {
			continue
		}
		v.Field(i).Set(build(d, alias{
			t:    field.Type,
			name: getAliasName(tag),
		}))
	}
}

func getTagMap(tag string) map[string]string {
	tagMap := make(map[string]string, 3)
	for _, part := range strings.Split(tag, ";") {
		kvs := strings.Split(part, ":")
		if len(kvs) == 1 {
			tagMap[kvs[0]] = ""
		} else {
			tagMap[kvs[0]] = kvs[1]
		}
	}
	return tagMap
}

func getAliasName(tag string) (name string) {
	tagMap := getTagMap(tag)
	return tagMap[diAlias]
}

func assert(pass bool, err error) {
	if !pass {
		panic(err)
	}
}

func (d *DI) Clean() {
	for _, v := range d.typeRepo {
		if _, ok := v.Interface().(*DI); ok {
			continue
		}
		if cleaner, ok := v.Interface().(Cleaner); ok {
			cleaner.Clean()
		}
	}
}
