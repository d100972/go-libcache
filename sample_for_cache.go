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
	if val, found, err := cacher.Get("key1"); err == nil {
		if found {
			fmt.Println("Found: ", val)
		} else {
			fmt.Println("Not found item.")
		}
	} else {
		fmt.Println("Error: ", err)
	}
	/*time.Sleep(time.Second * 10)

	if val, found := cacher.Get("key1"); found {
		fmt.Println("Found: ", val)
	} else {
		fmt.Println("Not found item.")
	}*/

	err := cacher.SaveMemToFile("")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	time.Sleep(time.Second * 3)
	err = cacher.LoadFileToMem("")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	if val, found, err := cacher.Get("key1"); err == nil {
		if found {
			fmt.Println("Found: ", val)
		} else {
			fmt.Println("Not found item.")
		}
	} else {
		fmt.Println("Error: ", err)
	}
	return
}
