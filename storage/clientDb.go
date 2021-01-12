////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for client-related functionality
//+build !stateless

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
)

// Inserts client registration code with given number of uses
func (d *DatabaseImpl) InsertClientRegCode(code string, uses int) error {
	jww.INFO.Printf("Inserting code %s, %d uses remaining", code, uses)
	return d.db.Create(&RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	}).Error
}

// If client registration code is valid, decrements remaining uses
func (d *DatabaseImpl) UseCode(code string) error {
	// Look up given registration code
	regCode := RegistrationCode{}
	jww.INFO.Printf("Attempting to use code %s...", code)
	err := d.db.First(&regCode, "code = ?", code).Error
	if err != nil {
		// Unable to find code, return error
		return err
	}

	if regCode.RemainingUses < 1 {
		// Code has no remaining uses, return error
		return errors.Errorf("Code %s has no remaining uses", code)
	}

	// Decrement remaining uses by one
	regCode.RemainingUses -= 1
	err = d.db.Save(&regCode).Error
	if err != nil {
		return err
	}

	jww.INFO.Printf("Code %s used, %d uses remaining", code,
		regCode.RemainingUses)
	return nil
}

// Gets User from the Database
func (d *DatabaseImpl) GetUser(publicKey string) (*User, error) {
	user := &User{}
	result := d.db.First(&user, "public_key = ?", publicKey)
	return user, result.Error
}

// Inserts User into the Database
func (d *DatabaseImpl) InsertUser(publicKey string) error {
	user := &User{
		PublicKey: publicKey,
	}
	return d.db.Create(user).Error
}

// Adds Client registration codes to the Database
func PopulateClientRegistrationCodes(codes []string, uses int) {
	for _, code := range codes {
		err := PermissioningDb.InsertClientRegCode(code, uses)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Client registration code: %+v",
				err)
		}
	}
}

// Adds Node registration codes to the Database
func PopulateNodeRegistrationCodes(infos []node.Info) {
	// TODO: This will eventually need to be updated to intake applications too
	i := 1
	for _, info := range infos {
		err := PermissioningDb.InsertApplication(&Application{
			Id: uint64(i),
		}, &Node{
			Code:          info.RegCode,
			Sequence:      info.Order,
			ApplicationId: uint64(i),
		})
		if err != nil {
			jww.ERROR.Printf("Unable to populate Node registration code: %+v",
				err)
		}
		i++
	}
}
