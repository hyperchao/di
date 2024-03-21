package di

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
)

const (
	diTag   = "di"
	diType  = "type"
	diAlias = "alias"
)

type Option func(*DI)

func Strict() func(d *DI) {
	return func(d *DI) {
		d.strict = true
	}
}

type DI struct {
	typeBuilder  map[reflect.Type]reflect.Value
	aliasBuilder map[alias]reflect.Value
	typeRepo     map[reflect.Type]*buildValue
	aliasRepo    map[alias]*buildValue
	cleaners     *list.List

	strict bool
}

type Cleaner interface {
	Clean()
}

type alias struct {
	t    reflect.Type
	name string
}

type buildValue struct {
	val  reflect.Value
	done bool
}

func New(opts ...Option) *DI {
	di := &DI{
		typeBuilder:  make(map[reflect.Type]reflect.Value),
		aliasBuilder: make(map[alias]reflect.Value),
		typeRepo:     make(map[reflect.Type]*buildValue),
		aliasRepo:    make(map[alias]*buildValue),
		cleaners:     list.New(),

		strict: false,
	}
	for _, opt := range opts {
		opt(di)
	}
	return di
}

func assert(pass bool, err error) {
	if !pass {
		panic(err)
	}
}

func (d *DI) Register(f any) {
	rt := reflect.TypeOf(f)
	assert(rt.Kind() == reflect.Func, fmt.Errorf("register typeBuilder of non func kind: %s", rt.Kind()))
	assert(rt.NumIn() == 0 && rt.NumOut() == 1, fmt.Errorf("typeBuilder func must have zero input ant one output"))

	t := rt.Out(0)
	_, ok := d.typeBuilder[t]
	assert(!ok, fmt.Errorf("duplicate register for type: %s", t.Name()))
	d.typeBuilder[t] = reflect.ValueOf(f)
}

func (d *DI) RegisterAlias(name string, f any) {
	rt := reflect.TypeOf(f)
	assert(rt.Kind() == reflect.Func, fmt.Errorf("register typeBuilder of non func kind: %s", rt.Kind()))
	assert(rt.NumIn() == 0 && rt.NumOut() == 1, fmt.Errorf("typeBuilder func must have zero input ant one output"))
	t := rt.Out(0)
	key := alias{
		t:    t,
		name: name,
	}
	_, ok := d.aliasBuilder[key]
	assert(!ok, fmt.Errorf("duplicate register for alias, name: %s, type: %s", name, t.Name()))
	d.aliasBuilder[key] = reflect.ValueOf(f)
}

func (d *DI) Build(p any) {
	rt := reflect.TypeOf(p)
	assert(rt.Kind() == reflect.Pointer, fmt.Errorf("build result must be pointer"))
	reflect.ValueOf(p).Elem().Set(d.build(rt.Elem()))
}

func (d *DI) Clean() {
	for c := d.cleaners.Front(); c != nil; c = c.Next() {
		c.Value.(Cleaner).Clean()
	}
}

func (d *DI) build(t reflect.Type) reflect.Value {
	v, ok := d.typeRepo[t]
	if ok {
		assert(v.done || !d.strict, fmt.Errorf("loop detected when build type: %s", t))
		return v.val
	}
	f := d.typeBuilder[t]
	assert(f.IsValid(), fmt.Errorf("typeBuilder absence for type: %s", t.String()))
	value := f.Call([]reflect.Value{})[0]
	v = &buildValue{val: value, done: false}
	d.typeRepo[t] = v
	d.buildStruct(value)
	v.done = true
	if cleaner, ok := value.Interface().(Cleaner); ok {
		d.cleaners.PushFront(cleaner)
	}
	return value
}

func (d *DI) buildAlias(a alias) reflect.Value {
	v, ok := d.aliasRepo[a]
	if ok {
		assert(v.done || !d.strict, fmt.Errorf("loop detected when build alias, name: %s, type: %s", a.name, a.t.String()))
		return v.val
	}
	f := d.aliasBuilder[a]
	assert(f.IsValid(), fmt.Errorf("typeBuilder absence for alias, name: %s, type: %s", a.name, a.t.String()))
	value := f.Call([]reflect.Value{})[0]
	v = &buildValue{val: value, done: false}
	d.aliasRepo[a] = v
	d.buildStruct(value)
	v.done = true
	if cleaner, ok := value.Interface().(Cleaner); ok {
		d.cleaners.PushFront(cleaner)
	}
	return value
}

func (d *DI) buildStruct(v reflect.Value) {
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
		name := d.getAliasName(tag)
		if name == "" {
			v.Field(i).Set(d.build(field.Type))
		} else {
			v.Field(i).Set(d.buildAlias(alias{
				t:    field.Type,
				name: name,
			}))
		}
	}
}

func (d *DI) getAliasName(tag string) (name string) {
	tagMap := make(map[string]string, 2)
	for _, part := range strings.Split(tag, ";") {
		kvs := strings.Split(part, ":")
		if len(kvs) == 1 {
			tagMap[kvs[0]] = ""
		} else {
			tagMap[kvs[0]] = kvs[1]
		}
	}
	return tagMap[diAlias]
}
