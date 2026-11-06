package conf

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

//go:embed locales/config.yaml
var configYAML []byte

var Config *Conf

type Conf struct {
	MySql map[string]*MySql `mapstructure:"mysql"`
	Redis *Redis            `mapstructure:"redis"`
	Oss   *Oss              `mapstructure:"oss"`
	I18n  *I18n             `mapstructure:"i18n"`
}

// I18n 国际化配置：切换默认语言只改 defaultLang。
type I18n struct {
	// DefaultLang 默认语言标签（如 zh-CN / en-US），必须命中 pkg/i18n 已注册语言。
	DefaultLang string `mapstructure:"defaultLang"`
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

// InitConfig 从编译期内嵌的 yaml 加载配置，再叠加环境变量覆盖。
func InitConfig() error {
	c, err := parseConfig(configYAML)
	if err != nil {
		return err
	}
	Config = c
	return nil
}

// parseConfig 解析 yaml → 应用环境变量覆盖 → 校验必填项。
// 每次调用使用独立的 viper 实例，避免全局状态，便于测试。
func parseConfig(raw []byte) (*Conf, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	var c Conf
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := c.applyEnvOverrides(); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

// applyEnvOverrides 用环境变量覆盖配置，让密码等敏感项可以不打进二进制。
// 命名规则：DEMO_<模块>_<字段>，多数据源按配置名插入一段，例如
//
//	DEMO_MYSQL_DEFAULT_PASSWORD / DEMO_MYSQL_DEFAULT_HOST
//	DEMO_REDIS_PASSWORD / DEMO_REDIS_HOST
//	DEMO_OSS_ACCESS_KEY_SECRET
func (c *Conf) applyEnvOverrides() error {
	for name, my := range c.MySql {
		if my == nil {
			continue
		}
		prefix := "DEMO_MYSQL_" + strings.ToUpper(name) + "_"
		override(&my.UserName, prefix+"USER")
		override(&my.Password, prefix+"PASSWORD")
		override(&my.DbHost, prefix+"HOST")
		override(&my.DbPort, prefix+"PORT")
		override(&my.DbName, prefix+"NAME")
	}

	var errs []error
	if r := c.Redis; r != nil {
		override(&r.RedisHost, "DEMO_REDIS_HOST")
		override(&r.RedisPort, "DEMO_REDIS_PORT")
		override(&r.RedisUsername, "DEMO_REDIS_USERNAME")
		override(&r.RedisPassword, "DEMO_REDIS_PASSWORD")
		if err := overrideInt(&r.RedisDbName, "DEMO_REDIS_DB"); err != nil {
			errs = append(errs, err)
		}
	}

	if o := c.Oss; o != nil {
		override(&o.AccessKeyId, "DEMO_OSS_ACCESS_KEY_ID")
		override(&o.AccessKeySecret, "DEMO_OSS_ACCESS_KEY_SECRET")
		override(&o.BucketName, "DEMO_OSS_BUCKET")
		override(&o.Endpoint, "DEMO_OSS_ENDPOINT")
	}

	return errors.Join(errs...)
}

// override 仅在环境变量存在时覆盖，未设置则沿用 yaml 中的值。
func override(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func overrideInt(dst *int, key string) error {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("环境变量 %s 不是合法整数: %q", key, v)
	}
	*dst = n
	return nil
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
