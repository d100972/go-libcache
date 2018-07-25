package main

import (
	"cache"
	"fmt"
	"time"
)

func main() {
	defaultExpiration, _ := time.ParseDuration("0.5h")
	gcInterval, _ := time.ParseDuration("5s")
	cacher := cache.NewCache(defaultExpiration, gcInterval)
	key1 := "hello world"
	expiration, _ := time.ParseDuration("6s")
	cacher.Set("key1", key1, expiration)
	if val, found := cacher.Get("key1"); found {
		fmt.Println("Found: ", val)
	} else {
		fmt.Println("Not found item.")
	}
	/*time.Sleep(time.Second * 10)

	if val, found := cacher.Get("key1"); found {
		fmt.Println("Found: ", val)
	} else {
		fmt.Println("Not found item.")
	}*/

	cacher.SaveMemToFile("/home/xj/Desktop/mycache.txt")
	time.Sleep(time.Second * 3)
	cacher.LoadFileToMem("/home/xj/Desktop/mycache.txt")
	if val, found := cacher.Get("key1"); found {
		fmt.Println("Found: ", val)
	} else {
		fmt.Println("Not found item.")
	}
	return
}
