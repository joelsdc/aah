// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package acrypto

import (
	"errors"
	"fmt"

	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
)

const hashDelim = "$"

var (
	// ErrPasswordEncoderIsNil returned when given password encoder instance is nil.
	ErrPasswordEncoderIsNil = errors.New("acrypto: password encoder is nil")

	passEncoders = make(map[string]PasswordEncoder)
)

// PasswordEncoder interface is used to generate password hash and compare given hash & password
// based chosen hashing type. Such as `bcrypt`, `scrypt` and `pbkdf2`.
//
// Good read about hashing security https://crackstation.net/hashing-security.htm
type PasswordEncoder interface {
	Generate(password []byte) ([]byte, error)
	Compare(hash, password []byte) bool
}

// PasswordAlgorithm method returns the password encoder for given algorithm,
// Otherwise nil. Out-of-the-box supported passowrd algorithms are `bcrypt`, `scrypt`
// and `pbkdf2`. You can add your own if need be via method `AddPasswordEncoder`.
func PasswordAlgorithm(alg string) PasswordEncoder {
	if pe, found := passEncoders[alg]; found {
		return pe
	}
	log.Warnf("acrypto: password algorithm '%s' not found", alg)
	return nil
}

// AddPasswordAlgorithm method is add password algorithm to encoders list.
// Implementation have to implement interface `PasswordEncoder`.
func AddPasswordAlgorithm(name string, pe PasswordEncoder) error {
	if pe == nil {
		return ErrPasswordEncoderIsNil
	}

	if _, found := passEncoders[name]; found {
		log.Warnf("acrypto: password encoder '%v' is already added", name)
		return nil
	}

	passEncoders[name] = pe

	return nil
}

// InitPasswordEncoders method initializes the password encoders based defined
// configuration in `security.password_encoder { ... }`
func InitPasswordEncoders(cfg *config.Config) error {
	keyPrefix := "security.password_encoder"

	// bcrypt algorithm
	if cfg.BoolDefault(keyPrefix+".bcrypt.enable", true) {
		bcryptCost := cfg.IntDefault("key", 12)
		if err := AddPasswordAlgorithm("bcrypt", &BcryptEncoder{cost: bcryptCost}); err != nil {
			return err
		}
	}

	// scrypt algorithm
	if cfg.BoolDefault(keyPrefix+".scrypt.enable", false) {
		n := cfg.IntDefault(keyPrefix+".scrypt.cpu_memory_cost", 32768)
		r := cfg.IntDefault(keyPrefix+".scrypt.block_size", 8)
		p := cfg.IntDefault(keyPrefix+".scrypt.parallelization", 1)
		dkLen := cfg.IntDefault(keyPrefix+".scrypt.derived_key_length", 32)
		saltLen := cfg.IntDefault(keyPrefix+".scrypt.salt_length", 24)

		if err := AddPasswordAlgorithm("scrypt", &ScryptEncoder{
			n: n, r: r, p: p, saltLen: saltLen, dkLen: dkLen}); err != nil {
			return err
		}
	}

	// pbkdf2 algorithm
	if cfg.BoolDefault(keyPrefix+".pbkdf2.enable", false) {
		iter := cfg.IntDefault(keyPrefix+".pbkdf2.iteration", 10000)
		dkLen := cfg.IntDefault(keyPrefix+".pbkdf2.derived_key_length", 32)
		saltLen := cfg.IntDefault(keyPrefix+".pbkdf2.salt_length", 24)
		hashAlg := cfg.StringDefault(keyPrefix+".pbkdf2.hash_algorithm", "sha-512")

		if hashFunc(hashAlg) == nil {
			return fmt.Errorf("acrypto/pbkdf2: invalid sha algorithm '%s'", hashAlg)
		}

		if err := AddPasswordAlgorithm("pbkdf2", &Pbkdf2Encoder{
			iter: iter, dkLen: dkLen, saltLen: saltLen, hashAlg: hashAlg}); err != nil {
			return err
		}
	}

	return nil
}
