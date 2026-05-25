package auth

import (
	"chat/utils"
	"github.com/spf13/viper"
)

type CertResponse struct {
	Status   bool `json:"status" required:"true"`
	Cert     bool `json:"cert"`
	Teenager bool `json:"teenager"`
}

func Cert(username string) *CertResponse {
	res, err := utils.Post(getDeeptrainApi("/app/cert"), map[string]string{
		"Content-Type": "application/json",
	}, map[string]interface{}{
		"password": viper.GetString("auth.access"),
		"user":     username,
		"hash":     utils.Sha2Encrypt(username + viper.GetString("auth.salt")),
	})

	if err != nil {
		return nil
	}

	resp, ok := decodeDeeptrainResponse[CertResponse](res)
	if !ok {
		return nil
	}
	return resp
}
