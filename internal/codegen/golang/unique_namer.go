package golang

import "fmt"

type UniqueNamer struct {
	used map[string]int
}

func NewUniqueNamer() *UniqueNamer {
	return &UniqueNamer{used: make(map[string]int)}
}

func (u *UniqueNamer) UniqueName(name string) string {
	if n, ok := u.used[name]; ok {
		newName := fmt.Sprintf("%s%d", name, n)
		u.used[name] += 1
		u.used[newName] = 1
		return newName
	} else {
		u.used[name] = 1
		return name
	}
}
