package utils

import (
	"github.com/pquerna/otp/totp"
)

func NewTOPT(issuer string, accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}
