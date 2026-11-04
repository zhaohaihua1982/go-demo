package conf

import (
	"os"

	"github.com/spf13/viper"
)

var Config *Conf

type Conf = struct {
	Email *Email `yaml:"email"`
}

type Email struct {
	Address   string `yaml:"address"`
	SmtpHost  string `yaml:"smtpHost"`
	SmtpEmail string `yaml:"smtpEmail"`
	SmtpPass  string `yaml:"smtpPass"`
}

func InitConfig() {
	workDir, _ := os.Getwd()
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(workDir + "/conf/locales")
	viper.SetConfigFile(workDir + "/conf/locales/config.yaml")
	_err := viper.ReadInConfig()
	if _err != nil {
		panic(_err)
	}

	// 把 yaml 内容反序列化到 Config，否则 Config 永远是 nil
	if err := viper.Unmarshal(&Config); err != nil {
		panic(err)
	}
}

// GetString 供外部直接读取 viper 中的配置项
func GetString(key string) string { return viper.GetString(key) }
