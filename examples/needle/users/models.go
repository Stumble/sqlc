// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0-70-g644434f9-wicked-fork

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
