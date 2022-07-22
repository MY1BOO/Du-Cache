package main

/*
$ curl "http://localhost:9999/api?key=Tom"
630

$ curl "http://localhost:9999/api?key=kkk"
kkk not exist
*/

import (
	"Du-Cache/DuCache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

// 本地数据源
// 如果在缓存中没有找到，就会通过回调函数访问这里的数据源
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建一个缓存空间Group，返回*geecache.Group
func createGroup() *DuCache.Group {
	// Group的名称是scores，缓存大小是2的10次方，回调函数是从上述本地数据源的map找对应的键值对
	return DuCache.NewGroup("scores", 2<<10, DuCache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器：创建HTTPPool，添加节点信息，注册到gee中，启动HTTP服务（共3个端口，8001/8002/8003），用户不感知
func startCacheServer(addr string, addrs []string, gee *DuCache.Group) {
	// 创建HTTPPool
	peers := DuCache.NewHttpPool(addr)
	// 添加节点信息
	peers.Set(addrs...)
	// 将节点注册到Group
	gee.RegisterPeers(peers)
	log.Println("Ducache is running at", addr)
	// 启动HTTP服务
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 开启api服务，与用户进行交互
func startAPIServer(apiAddr string, gee *DuCache.Group) {
	// 第一个参数是路由匹配规则，第二个参数是调用接口型函数HandlerFunc，传入一个处理请求的方法
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// 解析Get请求的URL，并找到匹配的值
			// 如http://localhost:9999/api?key=Tom解析找到"key"对应的是Tom，返回Tom
			key := r.URL.Query().Get("key")
			// 从缓存空间Group找key对应的缓存，返回找到的缓存值
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 设置HTTP头
			w.Header().Set("Content-Type", "application/octet-stream")
			// 响应http请求并传输找到的缓存值
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	// 开启监听服务，第二个参数不用指定是因为上面http.Handle已经指定了请求处理逻辑
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	/*
		使用flag包，解析命令行参数，分为两步：
		（1）绑定；（2）解析。
		使用效果：
			go build -o server
			./server -port=8003 -api=1
		变量port会被赋值为8003，api赋值为1
	*/
	// 将命令行中的port（第二个参数）绑定在变量port（第一个参数）上，默认值是8001，usage是帮助信息
	flag.IntVar(&port, "port", 8001, "Ducache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	// 把用户传递的命令行参数解析为对应变量的值
	flag.Parse()

	// 本节点开启节点服务的地址和端口
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// 取map中的所有value，放入切片中
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建一个缓存空间Group，返回*geecache.Group
	gee := createGroup()
	// 若api是true，开启api服务，用户可通过端口9999进行访问
	if api {
		go startAPIServer(apiAddr, gee)
	}
	// 启动缓存服务器
	// addrs的值是["http://localhost:8001","http://localhost:8002","http://localhost:8003"]
	startCacheServer(addrMap[port], addrs, gee)
}
