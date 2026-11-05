package test

import (
	"fmt"
	"testing"
)

func deferAdd(a, b int) int {
	t := a + b
	defer func() {
		a += t
		fmt.Println("a=", a)
	}()
	return t
}

func TestDeferAdd(*testing.T) {
	deferAdd(1, 2)
}

// 写一个方法测试 defer 的执行顺序
func TestDeferDo(*testing.T) {
	defer func() {
		fmt.Println("defer1")
	}()
	defer func() {
		fmt.Println("defer2")
	}()
	defer func() {
		fmt.Println("defer3")
	}()
	fmt.Println("end")
}
