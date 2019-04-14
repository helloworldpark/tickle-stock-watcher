package commons

import (
	"fmt"
	"testing"
)

func TestConcurrentMap(t *testing.T) {
	m := NewConcurrentMap()

	a := make(chan bool)
	go func() {
		v, ok := m.GetValue(8)
		fmt.Printf("Value: %v, OK: %v\n", v, ok)
		m.SetValue(8, 5)
		m.SetValue(1, 1)
		m.SetValue(3, 2)
		m.SetValue(5, 3)
		m.SetValue(7, 5)

		a <- true
	}()
	b := make(chan bool)
	go func() {
		m.SetValue(2, 1)
		m.SetValue(4, 2)
		m.SetValue(6, 3)
		v, ok := m.GetValue(8)
		fmt.Printf("Value: %v, OK: %v\n", v, ok)
		m.SetValue(8, 7)
		b <- true
	}()

	aa := <-a
	bb := <-b

	if aa && bb {
		fmt.Println(m.Count())
		m.Iterate(func(k, v interface{}, stop *bool) {
			fmt.Println(k, v)
			if k.(int) == 2 {
				*stop = true
			}
		})
	}
}
