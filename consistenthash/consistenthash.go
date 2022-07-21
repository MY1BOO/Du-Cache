package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//允许用户自定义hash函数
type Hash func(data []byte) uint32

type Map struct {
	//hash函数
	hash Hash
	//hash环
	keys []int
	//虚拟节点个数
	replicas int
	//虚拟节点和真实节点的映射
	hashmap map[int]string
}

func New(replicas int, hash Hash) *Map {
	m := &Map{
		hash:     hash,
		replicas: replicas,
		hashmap:  make(map[int]string),
	}
	//如果用户未自定义，默认使用
	if hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//根据真实节点添加虚拟节点
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//虚拟节点hash值入环
			m.keys = append(m.keys, hash)
			//添加虚拟节点hash值到真实节点名称的映射
			m.hashmap[hash] = key
		}
	}
	//排序
	sort.Ints(m.keys)
}
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	//二分查找环中第一个比当前hash值大的hash值下标
	index := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//从map中获取真实节点名称，下标取余数因为是个环
	return m.hashmap[m.keys[index%len(m.keys)]]
}
