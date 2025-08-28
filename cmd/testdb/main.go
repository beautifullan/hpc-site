package main

import (
	"fmt"
	"hpc-site/pkg" // 注意这里要换成你项目的 module 名
)

func main() {
	pkg.InitDB()
	defer pkg.DB.Close()

	fmt.Println("✅ 数据库连接成功！")

	// 试着跑一个简单查询
	var now string
	err := pkg.DB.Get(&now, "SELECT NOW()")
	if err != nil {
		panic(err)
	}
	fmt.Println("数据库时间:", now)
}
