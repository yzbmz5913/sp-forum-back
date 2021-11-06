package service

import (
	"reflect"
	"sync"
)

type services struct {
	Us *UserService
	Ts *ThreadService
	Hs *HomeService
}

var s *services
var once sync.Once

func S() *services {
	once.Do(func() {
		s = new(services)
		t := reflect.TypeOf(s).Elem()
		for i := 0; i < t.NumField(); i++ {
			reflect.ValueOf(s).Elem().Field(i).Set(reflect.New(t.Field(i).Type.Elem()))
		}
	})
	return s
}
