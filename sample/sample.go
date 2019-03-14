package main

import (
	"time"
	"cache"
	"fmt"
)

func main()  {
	defaultExpiration, _ := time.ParseDuration("0.5h")
	gcInterval, _ := time.ParseDuration("3s")
	c := cache.NewCache(defaultExpiration, gcInterval)

	k1 := "hello ,I am pasca"
	expiration, _ :=time.ParseDuration("5s")

	c.Set("k1", k1,  expiration)
	s, _:=time.ParseDuration("10s")
	if v, found :=c.Get("k1"); found{
		fmt.Println("found k1:", v)
	}else {
		fmt.Println("not found k1")
	}
	// 暂停10s
	time.Sleep(s)
	//现在K1应该被清理了
	if v, found :=c.Get("k1"); found {
		fmt.Println("Found k1:", v)
	}else {
		fmt.Println("not found k1")
	}

}