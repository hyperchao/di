package di

import (
	"fmt"
	"testing"
)

type ServiceA interface {
	AAA()
}

type ServiceB interface {
	BBB()
}

type serviceAImpl struct {
	ServiceB       ServiceB       `di:"type"`
	ServiceB2      ServiceB       `di:"type"`
	ServiceB3      *serviceBImpl3 `di:"type"`
	ServiceBAlias  ServiceB       `di:"alias:xxx"`
	ServiceBAlias2 ServiceB       `di:"alias:xxx"`
}

func NewServiceA() ServiceA {
	return &serviceAImpl{}
}

func (s *serviceAImpl) AAA() {
	fmt.Println("serviceAImpl AAA")
	s.ServiceB.BBB()
	s.ServiceB2.BBB()
	s.ServiceB3.BBB()
	s.ServiceBAlias.BBB()
	s.ServiceBAlias2.BBB()
}

func NewServiceB() ServiceB {
	fmt.Println("new serviceBImpl1")
	return &serviceBImpl1{}
}

type serviceBImpl1 struct {
}

func (s *serviceBImpl1) BBB() {
	fmt.Println("serviceBImpl1 BBB")
}

func NewServiceB2() ServiceB {
	fmt.Println("new serviceBImpl2")
	return &serviceBImpl2{}
}

type serviceBImpl2 struct {
}

func (s *serviceBImpl2) BBB() {
	fmt.Println("serviceBImpl2 BBB")
}

func (s *serviceBImpl2) Clean() {
	fmt.Println("serviceBImpl2 clean")
}

func NewServiceBImpl() *serviceBImpl3 {
	fmt.Println("new serviceBImpl3")
	return &serviceBImpl3{}
}

type serviceBImpl3 struct {
	ServiceA `di:"type"`
}

func (s *serviceBImpl3) BBB() {
	fmt.Println("serviceBImpl3 BBB")
}

func TestDI(t *testing.T) {
	di := New()
	RegisterDI(di, NewServiceA)
	RegisterDI(di, NewServiceB)
	RegisterDI(di, NewServiceBImpl)
	RegisterAliasDI(di, "xxx", NewServiceB2)
	s := GetDI[ServiceA](di)
	s.AAA()
	di.Clean()
}
