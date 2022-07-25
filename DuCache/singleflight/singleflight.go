package singleflight

import "sync"

//正在执行的函数结果的封装
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//主结构，将多个key的函数结果存到m中
type Group struct {
	//保护m不会被并发读写
	mutex sync.Mutex
	m     map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mutex.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//如果有已经在执行的key
	if c, ok := g.m[key]; ok {
		g.mutex.Unlock()
		//当前协程等待返回结果
		c.wg.Wait()
		return c.val, c.err
	}
	//将当前key存到m中
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mutex.Unlock()
	//执行查询
	c.val, c.err = fn()
	//唤醒在等待的协程
	c.wg.Done()
	//删除当前key
	g.mutex.Lock()
	delete(g.m, key)
	g.mutex.Unlock()

	return c.val, c.err
}
