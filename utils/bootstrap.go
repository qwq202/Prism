package utils

import (
	"chat/globals"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func ReadConf() {
	viper.SetConfigFile(configFile)

	if !IsFileExist(configFile) {
		fmt.Printf("[service] config.yaml not found, creating one from template: %s\n", configExampleFile)
		if err := CopyFile(configExampleFile, configFile); err != nil {
			fmt.Println(err)
		} else if err := generateSecretForNewConfig(configFile); err != nil {
			globals.Warn(fmt.Sprintf("[service] failed to generate secret for new config: %s", err.Error()))
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	secret := viper.GetString("secret")
	if isWeakSecret(secret) {
		globals.Warn(fmt.Sprintf("[service] weak secret: got %d bytes, expected at least 32 random bytes and not a placeholder; starting in 10 seconds, please set a stronger `secret` in config or environment; future versions may panic on weak secrets", len(secret)))
		time.Sleep(10 * time.Second)
	}

	if timeout := viper.GetInt("max_timeout"); timeout > 0 {
		globals.HttpMaxTimeout = time.Second * time.Duration(timeout)
		globals.Debug(fmt.Sprintf("[service] http client timeout set to %ds from env", timeout))
	}
}

func isWeakSecret(secret string) bool {
	value := strings.TrimSpace(secret)
	lower := strings.ToLower(value)
	return len(value) < 32 || lower == "secret" || strings.Contains(lower, "replace_with")
}

func generateSecretForNewConfig(path string) error {
	conf := viper.New()
	conf.SetConfigFile(path)
	if err := conf.ReadInConfig(); err != nil {
		return err
	}

	if !isWeakSecret(conf.GetString("secret")) {
		return nil
	}

	conf.Set("secret", GenerateChar(64))
	return conf.WriteConfig()
}

func NewEngine() *gin.Engine {
	if viper.GetBool("debug") {
		return gin.Default()
	}

	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	return engine
}
