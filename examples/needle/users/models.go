// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v2.0.2-dirty-wicked-fork

package users

import (
	"time"
)

type User struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	Metadata  []byte    `json:"metadata"`
	Thumbnail string    `json:"thumbnail"`
	CreatedAt time.Time `json:"createdat"`
}
