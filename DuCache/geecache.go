package DuCache

import (
	pb "Du-Cache/DuCache/ducachepb"
	"Du-Cache/DuCache/singleflight"
	"fmt"
	"log"
	"sync"
)

var (
	//读写锁
	mutex sync.RWMutex
	//全局groups
	groups = make(map[string]*Group)
)

//命名空间
type Group struct {
	//名称
	name string
	//回调函数
	getter    Getter
	mainCache cache
	// HttpPool对象（HTTP服务端），它实现了PeerPicker
	// 记录可访问的远程节点
	peers PeerPicker
	//保证同一时刻只有一个协程在执行查询，防止缓存击穿
	loader *singleflight.Group
}

//回调Getter，在缓存不存在时，调用这个函数，得到源数据
//这是一个接口型函数的实现
//函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入相同类型的函数作为参数，也能够传入实现了该接口的结构体作为参数
//定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//新建命名空间，并加入全局groups
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("Getter is nil!")
	}
	mutex.Lock()
	defer mutex.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//获取一个group，不存在共享变量的写操作所以加读锁
func GetGroup(name string) *Group {
	mutex.RLock()
	defer mutex.RUnlock()
	g := groups[name]
	return g
}

//从group中取数据，如果缓存命中返回数据，未命中调用load()
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if value, ok := g.mainCache.get(key); ok {
		log.Println("[Du-cache hit]--->" + value.String())
		return value, nil
	}
	return g.load(key)
}

//如果一致性哈希选择到了远程节点，则调用getFromPeer()从远程获取数据，否则调用getLocally()从本地获取数据
func (g *Group) load(key string) (value ByteView, err error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peerGetter, ok := g.peers.PickPeer(key); ok {
				byteView, err := g.getFromPeer(peerGetter, key)
				if err == nil {
					return byteView, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return
}

//调用用户设置的回调函数Getter从本地获取数据，然后再调用populateCache()将数据放入缓存
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	byteview := ByteView{cloneBytes(bytes)}
	g.populateCache(key, byteview)
	return byteview, nil
}

//将数据放入缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

//从远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
