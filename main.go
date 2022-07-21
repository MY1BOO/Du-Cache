package main

import (
	"Du-Cache/group"
	duhttp "Du-Cache/http"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":   "630",
	"Jack":  "589",
	"Sam":   "567",
	"嘟是什么嘟": "哈纸嘟",
}

func main() {
	group.NewGroup("scores", 2<<10, group.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:9999"
	peers := duhttp.NewHttpPool(addr)
	log.Println("Du-Cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
