package main

import (
	"github.com/plutolove233/co-dream/internal/setting"
	"github.com/plutolove233/co-dream/internal/utils/rsa"
	"github.com/spf13/viper"
)

func main() {
	_ = setting.InitViper()
	x := rsa.RSA{
		PublicKeyPath:  viper.GetString("system.RSAPublic"),
		PrivateKeyPath: viper.GetString("system.RSAPrivate"),
	}
	x.GenerateRSAKey(2048)
}
