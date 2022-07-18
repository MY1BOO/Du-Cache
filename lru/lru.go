package lru

import "container/list"

type Cache struct {
	// 允许使用的最大内存
	maxBytes int64
	// 当前已使用的内存
	nbytes int64
	// 双向链表,每个节点是一个Element，里面包含着上下两个Element指针和一个空接口类型的Value值
	ll *list.List
	// 键是字符串，值是双向链表的对应节点指针
	cache map[string]*list.Element
	// 某条记录被移除时的回调函数，可为nil
	OnEvicted func(key string, value Value)
}

// entry表示双向链表节点的数据类型
// 在链表中仍保存每个值对应的key的好处在于，淘汰队首节点时，需要用key从字典中删除对应的映射
type entry struct {
	key string
	// 值是实现了Value接口的任意类型，该接口只包含了一个方法 Len() int，用于返回“值”所占用的内存大小。
	value Value
}

// Value接口的len方法返回“值”所占用的内存大小
type Value interface {
	Len() int
}

//构造函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//添加元素
func (c *Cache) Add(key string, value Value) {
	//若key存在则将节点移到队头并更新key所对应的value值
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		//将节点中的值断言成entry类型指针
		kv := element.Value.(*entry)
		//重新计算已使用的内存
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		//更新
		kv.value = value
	} else {
		element := c.ll.PushFront(&entry{key, value})
		c.cache[key] = element
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 更新c.nbytes，如果超过了设定的最大值c.maxBytes，则移除最少访问的节点
	// 当maxBytes等于0时，不触发，即默认无限添加
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

//查找元素
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		return kv.value, true
	}
	return
}

//删除元素
func (c *Cache) RemoveOldest() {
	element := c.ll.Back()
	if element != nil {
		//从双向链表中删除
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		//从cache中删除
		delete(c.cache, kv.key)
		//更新占用内存
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//触发回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

//获取cache元素个数
func (c *Cache) Len() int {
	return c.ll.Len()
}
