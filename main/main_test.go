package main

import (
	"fmt"
	"testing"
	"time"
)

func TestSlice(t *testing.T) {
	foo := []int{0, 0, 0, 42, 100}
	bar := foo[1:4]
	bar[1] = 99
	fmt.Println("foo:", foo) // [0,0,99,42,100]
	fmt.Println("bar:", bar) // [0,99,42]

	x := []int{1, 2, 3, 4}
	y := x[:2]
	fmt.Println(cap(x), cap(y)) // 4 4
	y = append(y, 30)
	fmt.Printf("x:%v %p\n", x, x) // [1,2,30,4]
	fmt.Printf("y:%v %p\n", y, y) // [1,2,30]

	// 容量有大小一盘是从切片有开始位置到底层数据的结尾位置的长度
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8}
	num1 := nums[2:4]
	num2 := nums[:3]
	num3 := nums[3:]
	fmt.Printf("num1 cap:%v ,nums2 cap:%v: nums3 cap:%v\n", cap(num1), cap(num2), cap(num3))

}

func TestCh(t *testing.T) {
	ticker := time.NewTicker(1 * time.Second)
	ticker2 := time.NewTicker(2 * time.Second)
	var ch = make(chan int)
	go func() {
		for range ticker2.C {
			close(ch)
		}
	}()
	for range ticker.C {
		select {
		case ch <- 12:
			fmt.Println("send succes")
		default:
			fmt.Println("send failed")
		}
	}
}

func TestMain(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5}
	for k, v := range arr {
		fmt.Printf("%v %v a1:%p a2:%p\n", k, v, &k, &v)

	}
}
