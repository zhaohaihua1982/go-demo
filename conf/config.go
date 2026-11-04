package conf

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

//go:embed locales/config.yaml
var configYAML []byte

var Config *Conf

type Conf struct {
	MySql map[string]*MySql `mapstructure:"mysql"`
	Redis *Redis            `mapstructure:"redis"`
	Oss   *Oss              `mapstructure:"oss"`
}

type Oss struct {
	BucketName      string `mapstructure:"bucketName"`
	AccessKeyId     string `mapstructure:"accessKeyId"`
	AccessKeySecret string `mapstructure:"accessKeySecret"`
	Endpoint        string `mapstructure:"endPoint"`
	EndpointOut     string `mapstructure:"endpointOut"`
	QiNiuServer     string `mapstructure:"qiNiuServer"`
}

type MySql struct {
	Dialect  string `mapstructure:"dialect"`
	DbHost   string `mapstructure:"dbHost"`
	DbPort   string `mapstructure:"dbPort"`
	DbName   string `mapstructure:"dbName"`
	UserName string `mapstructure:"userName"`
	Password string `mapstructure:"password"`
	Charset  string `mapstructure:"charset"`
}

type Redis struct {
	RedisHost     string `mapstructure:"redisHost"`
	RedisPort     string `mapstructure:"redisPort"`
	RedisUsername string `mapstructure:"redisUsername"`
	RedisPassword string `mapstructure:"redisPwd"`
	RedisDbName   int    `mapstructure:"redisDbName"`
	RedisNetwork  string `mapstructure:"redisNetwork"`
}

// InitConfig 从编译期内嵌的 yaml 加载配置，不依赖进程工作目录。
func InitConfig() error {
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewReader(configYAML)); err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	var c Conf
	if err := viper.Unmarshal(&c); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	Config = &c

	return Config.Validate()
}

// Validate 校验必填项，避免启动后拿到空地址再报 nil/连接错误。
func (c *Conf) Validate() error {
	var errs []error

	my := c.MySql["default"]
	if my == nil {
		errs = append(errs, errors.New("mysql.default 未配置"))
	} else {
		if my.DbHost == "" {
			errs = append(errs, errors.New("mysql.default.dbHost 不能为空"))
		}
		if my.DbPort == "" {
			errs = append(errs, errors.New("mysql.default.dbPort 不能为空"))
		}
		if my.DbName == "" {
			errs = append(errs, errors.New("mysql.default.dbName 不能为空"))
		}
	}

	if c.Redis == nil {
		errs = append(errs, errors.New("redis 未配置"))
	} else {
		if c.Redis.RedisHost == "" {
			errs = append(errs, errors.New("redis.redisHost 不能为空"))
		}
		if c.Redis.RedisPort == "" {
			errs = append(errs, errors.New("redis.redisPort 不能为空"))
		}
	}

	return errors.Join(errs...)
}

// GetString 供外部直接读取 viper 中的配置项
func GetString(key string) string { return viper.GetString(key) }
