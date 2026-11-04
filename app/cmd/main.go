package main

import (
	"demo/app/biz/infrastructure/cache"
	dao "demo/app/biz/infrastructure/db"
	"encoding/json"
	"fmt"

	"demo/conf"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>
func main() {

	initEnvConfig()

	// 方式1：直接打印结构体指针，%+v 会带上字段名
	fmt.Printf("1) %+v\n", conf.Config)

	// 方式2：打印某个字段
	fmt.Printf("2) DbHost = %s\n", conf.Config.Redis.RedisHost)

	// 方式3：转成 JSON 缩进输出，最直观
	b, _ := json.MarshalIndent(conf.Config, "", "  ")
	fmt.Printf("3) %s\n", b)

	// 方式4：绕过结构体，直接从 viper 取
	fmt.Printf("4) viper: %s / %s\n", conf.GetString("redis.redisHost"), conf.GetString("redis.redisPort"))

	cache.RedisClient.Set(cache.RedisContext, "name", "dugu", 0)

	name, _ := cache.RedisClient.Get(cache.RedisContext, "name").Result()
	fmt.Printf("5) Redis: %s\n", name)
}

func initEnvConfig() {
	conf.InitConfig()
	dao.InitMySQL()
	cache.InitCache()
}
