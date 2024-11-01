package g

import (
	"fmt"
	"testing"
)

func TestNewGetSet(t *testing.T) {
	m := NewMemory[any](10)
	fmt.Println(m.Index("a123456", m.Count()))
	fmt.Println(m.Index("a123123", m.Count()))
	fmt.Println(m.Index("a112233", m.Count()))
	m.Set("abc", "123")
	value, ok := m.Get("abc")
	fmt.Println(value, ok)
}
