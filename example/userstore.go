package main

import (
	"github.com/patrickmn/go-cache"
)

type UserStore struct {
	cache *cache.Cache
}

type User struct {
	TGID  string
	State string
}

func NewUserStore() *UserStore {
	return &UserStore{
		cache: cache.New(-1, -1),
	}
}

func (store *UserStore) Get(tgID string) (*User, bool) {
	usrMap, ok := store.cache.Get(tgID)
	if ok {
		return usrMap.(*User), ok
	}

	return nil, false
}

func (store *UserStore) Store(user *User) {
	store.cache.Set(user.TGID, user, -1)
}
