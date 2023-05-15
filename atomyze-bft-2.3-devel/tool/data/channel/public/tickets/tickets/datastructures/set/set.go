package set

import (
	"fmt"
)

type Set map[string]struct{}

func (s Set) Add(el string) {
	s[el] = struct{}{}
}

func (s Set) Delete(el string) {
	if _, ok := s[el]; ok {
		delete(s, el)
	}
}

func (s Set) Contains(el string) (ok bool) {
	_, ok = s[el]
	return
}

func (s Set) ToSlice() []string {
	sl := make([]string, 0, len(s))
	for k, _ := range s {
		sl = append(sl, k)
	}

	return sl
}

func (s Set) Intersection(other Set) Set {
	if other == nil {
		return nil
	}
	intersection := Set{}
	for k, _ := range s {
		if other.Contains(k) {
			intersection.Add(k)
		}
	}
	return intersection
}

func (s Set) Difference(other Set) Set {
	if other == nil {
		return s
	}
	difference := Set{}
	for k, _ := range s {
		if !other.Contains(k) {
			difference.Add(k)
		}
	}
	return difference
}

func (s Set) String() string {
	return fmt.Sprintf("%v", s.ToSlice())
}

func FromSlice(slice []string) Set {
	if slice == nil {
		return nil
	}
	s := Set{}
	for _, el := range slice {
		s.Add(el)
	}
	return s
}
