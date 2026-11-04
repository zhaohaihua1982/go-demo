package conf

import (
	"os"

	"github.com/spf13/viper"
)

var Config *Conf

type Conf = struct {
	MySql map[string]*MySql `yaml:"mysql"`
	Redis *Redis            `yaml:"redis"`
	Oss   *Oss              `yaml:"oss"`
}

type Oss struct {
	BucketName      string `yaml:"bucketName"`
	AccessKeyId     string `yaml:"accessKeyId"`
	AccessKeySecret string `yaml:"accessKeySecret"`
	Endpoint        string `yaml:"endPoint"`
	EndpointOut     string `yaml:"endpointOut"`
	QiNiuServer     string `yaml:"qiNiuServer"`
}

type MySql struct {
	Dialect  string `yaml:"dialect"`
	DbHost   string `yaml:"dbHost"`
	DbPort   string `yaml:"dbPort"`
	DbName   string `yaml:"dbName"`
	UserName string `yaml:"userName"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
}

type Redis struct {
	RedisHost     string `yaml:"redisHost"`
	RedisPort     string `yaml:"redisPort"`
	RedisUsername string `yaml:"redisUsername"`
	RedisPassword string `yaml:"redisPwd"`
	RedisDbName   int    `yaml:"redisDbName"`
	RedisNetwork  string `yaml:"redisNetwork"`
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
