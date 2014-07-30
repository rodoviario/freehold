// Copyright 2014 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package user

import (
	"encoding/json"
	"errors"

	"bitbucket.org/tshannon/freehold/data"
	"bitbucket.org/tshannon/freehold/fail"
	"bitbucket.org/tshannon/freehold/log"
	"bitbucket.org/tshannon/freehold/setting"

	"code.google.com/p/go.crypto/bcrypt"
)

const (
	DS          = "core/user.ds"
	authLogType = "authentication"
)

var FailLogon = errors.New("Invalid user and / or password")

type User struct {
	Name        string `json:"name,omitempty"`
	Password    string `json:"password,omitempty"`
	EncPassword []byte `json:"encPassword,omitempty"`
	HomeApp     string `json:"homeApp,omitempty"`
	Admin       bool   `json:"admin,omitempty"`

	username    string `json:"-"`
	passwordSet bool   `json:"-"` //True when password is getting updated. Can only be set internally
	adminSet    bool   `json:"-"`
}

func Get(username string) (*User, error) {
	ds, err := data.OpenCoreDS(DS)
	if err != nil {
		return nil, err
	}
	key, err := json.Marshal(username)
	if err != nil {
		return nil, err
	}

	usr := &User{}

	value, err := ds.Get(key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fail.NewFromErr(FailLogon, username)
	}

	err = json.Unmarshal(value, usr)
	if err != nil {
		return nil, err
	}
	usr.username = username

	return usr, nil
}

func All() (map[string]*User, error) {
	ds, err := data.OpenCoreDS(DS)
	if err != nil {
		return nil, err
	}

	iter, err := ds.Iter(nil, nil)
	if err != nil {
		return nil, err
	}

	users := make(map[string]*User)

	for iter.Next() {
		if iter.Err() != nil {
			return nil, iter.Err()
		}

		usr := &User{}

		err = json.Unmarshal(iter.Value(), usr)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(iter.Key(), &usr.username)
		if err != nil {
			return nil, err
		}

		users[usr.username] = usr
	}

	return users, nil
}

func New(username string, u *User) error {
	u.username = username
	//Check if user exists
	ds, err := data.OpenCoreDS(DS)
	if err != nil {
		return err
	}
	key, err := json.Marshal(u.username)
	if err != nil {
		return err
	}

	if u.HomeApp == "" {
		u.HomeApp = setting.String("DefaultHomeApp")
	}

	value, err := ds.Get(key)
	if err != nil {
		return err
	}
	if value != nil {
		u.ClearPassword()
		return fail.New("User already exists", u)
	}

	err = u.setPassword(u.Password)
	if err != nil {
		return err
	}
	return u.Update()
}

func Delete(username string) error {
	ds, err := data.OpenCoreDS(DS)
	if err != nil {
		return err
	}
	key, err := json.Marshal(username)
	if err != nil {
		return err
	}

	err = ds.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Update() error {
	if u.username == "" {
		return fail.New("Invalid username", u.username)
	}

	if !u.passwordSet {
		u.ClearPassword()
	}

	ds, err := data.OpenCoreDS(DS)
	if err != nil {
		return err
	}
	key, err := json.Marshal(u.username)
	if err != nil {
		return err
	}

	value, err := json.Marshal(u)
	if err != nil {
		return err
	}

	err = ds.Put(key, value)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) SetAdmin(isAdmin bool) error {
	u.Admin = isAdmin
	u.adminSet = true
	//TODO: Log Admin changes
	return u.Update()
}

func (u *User) UpdatePassword(password string) error {
	err := u.setPassword(password)
	if err != nil {
		return err
	}

	err = u.Update()
	if err != nil {
		return err
	}

	if setting.Bool("LogPasswordChange") {
		log.NewEntry(authLogType, "User "+u.username+" has changed their password.")
	}
	return nil
}

func (u *User) setPassword(password string) error {
	if len(password) < setting.Int("MinPasswordLength") {
		u.ClearPassword()
		return fail.New("Password isn't long enough.", u)
	}
	encPass, err := bcrypt.GenerateFromPassword([]byte(password), setting.Int("PasswordBcryptWorkFactor"))
	if err != nil {
		return err
	}
	u.EncPassword = encPass
	u.Password = ""
	u.passwordSet = true

	return nil
}

func (u *User) Login(password string) error {
	if u.username == "" {
		return loginFailure(u)
	}

	//TODO: Rate limit login attempts? Or rate limit all public requests?
	if len(password) < setting.Int("MinPasswordLength") {
		return loginFailure(u)
	}

	err := bcrypt.CompareHashAndPassword(u.EncPassword, []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return loginFailure(u)
		}
		return err
	}
	if setting.Bool("LogSuccessAuth") {
		log.NewEntry(authLogType, "User "+u.username+" has logged in successfully.")
	}
	//done with password, clear it
	u.ClearPassword()

	return nil
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ClearPassword() {
	u.Password = ""
	u.EncPassword = nil
}

func loginFailure(u *User) error {
	if setting.Bool("LogFailedAuth") {
		log.NewEntry(authLogType, "User "+u.username+" has failed a login attempt.")
	}
	u.ClearPassword()
	return fail.NewFromErr(FailLogon, u.username)

}
