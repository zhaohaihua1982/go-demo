package conf

import (
	"strings"
	"testing"
)

// 测试用的最小 yaml，字段名与 conf/locales/config.yaml 保持一致
const testYAML = `
mysql:
  default:
    dialect: "mysql"
    dbHost: "10.0.0.1"
    dbPort: "3306"
    dbName: "mall_db"
    userName: "root"
    password: "from-yaml"
    charset: "utf8mb4"
oss:
  accessKeyId: "ak"
  accessKeySecret: "sk"
  bucketName: "bucket"
  endPoint: "oss.example.com"
redis:
  redisDbName: 4
  redisHost: "10.0.0.2"
  redisPort: 6379
  redisNetwork: "tcp"
  redisPwd: "redis-yaml"
`

func TestParseConfig_MapsAllFields(t *testing.T) {
	c, err := parseConfig([]byte(testYAML))
	if err != nil {
		t.Fatalf("parseConfig 失败: %v", err)
	}

	my := c.MySql["default"]
	if my == nil {
		t.Fatal("mysql.default 未解析")
	}
	if my.Password != "from-yaml" {
		t.Errorf("mysql 密码 = %q, want %q", my.Password, "from-yaml")
	}
	if my.DbHost != "10.0.0.1" || my.DbPort != "3306" || my.DbName != "mall_db" {
		t.Errorf("mysql 地址解析异常: %+v", my)
	}

	// 回归点：yaml 里的 redisPwd 必须映射到 RedisPassword（曾因 tag 用错而静默为空）
	if c.Redis == nil {
		t.Fatal("redis 未解析")
	}
	if c.Redis.RedisPassword != "redis-yaml" {
		t.Errorf("redis 密码 = %q, want %q（检查 mapstructure tag）", c.Redis.RedisPassword, "redis-yaml")
	}
	if c.Redis.RedisPort != "6379" {
		t.Errorf("redis 端口 = %q, want %q（int 应可弱转成 string）", c.Redis.RedisPort, "6379")
	}
	if c.Redis.RedisDbName != 4 {
		t.Errorf("redis db = %d, want 4", c.Redis.RedisDbName)
	}

	if c.Oss == nil || c.Oss.AccessKeySecret != "sk" || c.Oss.Endpoint != "oss.example.com" {
		t.Errorf("oss 解析异常: %+v", c.Oss)
	}
}

func TestParseConfig_EnvOverridesSecrets(t *testing.T) {
	t.Setenv("DEMO_MYSQL_DEFAULT_PASSWORD", "mysql-env")
	t.Setenv("DEMO_MYSQL_DEFAULT_HOST", "127.0.0.9")
	t.Setenv("DEMO_REDIS_PASSWORD", "redis-env")
	t.Setenv("DEMO_REDIS_DB", "7")
	t.Setenv("DEMO_OSS_ACCESS_KEY_SECRET", "sk-env")

	c, err := parseConfig([]byte(testYAML))
	if err != nil {
		t.Fatalf("parseConfig 失败: %v", err)
	}

	if got := c.MySql["default"].Password; got != "mysql-env" {
		t.Errorf("mysql 密码未被环境变量覆盖: %q", got)
	}
	if got := c.MySql["default"].DbHost; got != "127.0.0.9" {
		t.Errorf("mysql host 未被环境变量覆盖: %q", got)
	}
	if got := c.MySql["default"].DbName; got != "mall_db" {
		t.Errorf("未设置的字段应沿用 yaml: %q", got)
	}
	if got := c.Redis.RedisPassword; got != "redis-env" {
		t.Errorf("redis 密码未被环境变量覆盖: %q", got)
	}
	if got := c.Redis.RedisDbName; got != 7 {
		t.Errorf("redis db 未被环境变量覆盖: %d", got)
	}
	if got := c.Oss.AccessKeySecret; got != "sk-env" {
		t.Errorf("oss secret 未被环境变量覆盖: %q", got)
	}
}

func TestParseConfig_InvalidRedisDB(t *testing.T) {
	t.Setenv("DEMO_REDIS_DB", "not-a-number")

	_, err := parseConfig([]byte(testYAML))
	if err == nil {
		t.Fatal("非法 DEMO_REDIS_DB 应返回错误")
	}
	if !strings.Contains(err.Error(), "DEMO_REDIS_DB") {
		t.Errorf("错误信息应包含变量名, got: %v", err)
	}
}

func TestValidate_MissingSections(t *testing.T) {
	err := (&Conf{}).Validate()
	if err == nil {
		t.Fatal("空配置应校验失败")
	}
	for _, want := range []string{"mysql.default 未配置", "redis 未配置"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("校验错误缺少 %q, got: %v", want, err)
		}
	}
}

func TestParseConfig_BrokenYAML(t *testing.T) {
	if _, err := parseConfig([]byte("mysql: [oops")); err == nil {
		t.Fatal("非法 yaml 应返回错误")
	}
}
