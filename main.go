package main

import (
	"github.com/astaxie/beego"
	_ "updateFileProducer/routers"
)

/*

* 每次输出时，通过比较文件变化，更新输出目录的文件

TODO：
1. 项目中有同名文件存在时，如何忽略其中一个
*/
func main() {
	beego.Run()
}
