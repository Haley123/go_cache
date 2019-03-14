package cache

import (
	"time"
	"sync"
	"fmt"
	"encoding/gob"
	"os"
	"io"
)

type Item struct {
	Object      interface{}  //数据项
	Expiration  int64      //生存时间
}

//判断是否过期，提供Expired方法，返回bool即为过期
func (item Item ) Expired() bool{
	if item.Expiration ==0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

const (
	NoExpiration time.Duration = -1   //没有过期的时间

	DefaultExpiration time.Duration = 0 //默认过期时间

)

type Cache struct {
	defaultExpiration time.Duration
	items            map[string]Item //缓存数据存储在Map中
	mu               sync.RWMutex    //读写锁
	gcInterval       time.Duration   //过期数据项清理周期
	stopGc            chan bool
}

func (c *Cache) gcLoop()  {
	ticker :=time.NewTicker(c.gcInterval)   //通过time.NewTicker() 方法创建的 ticker, 会通过指定的c.Interval 间隔时间，周期性的从 ticker.C 管道中发送数据过来
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()  //然后周期性的执行删除
		case <-c.stopGc:
			ticker.Stop()   //监听C.stopGc管道，有数据即停止gcLoop的运行
		return
		}

	}


}
//删除缓存过期项
func (c *Cache) delete(k string)  {
	delete(c.items, k)
}


//删除过期数据项
func (c *Cache) DeleteExpired()  {
	now :=time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k,v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			c.delete(k)
		}

	}
}

//设置缓存数据项，如果数据存在则覆盖
func (c *Cache) Set(k string, v interface{}, d time.Duration) {
	var e int64
	if d == DefaultExpiration{
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = Item{
		Object:  v,
		Expiration: e,
	}

}
//获取数据项，如果找到数据项，还需要判断该数据项是否已经过期
func (c *Cache) get(k string) (interface{}, bool) {
	item, found :=c.items[k]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false

	}
	return item.Object, true
}

//添加数据项，如果想巨响已经存在，则返回错误
func (c *Cache) Add(k string, v interface{}, d time.Duration) error  {
	c.mu.Lock()
	_, found :=c.get(k)
	if found {
		c.mu.Unlock()
		return fmt.Errorf("item %s already exists", k)
	}
	c.Set(k, v, d)
	c.mu.Unlock()
	return nil
}

//获取数据项
func (c *Cache) Get(k string) (interface{}, bool)  {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found :=c.items[k]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}
	return item.Object, true
}

//替换一个㛮的数据项
func (c *Cache) Replace(k string, v interface{}, d time.Duration) error  {
	c.mu.Lock()
	_, found :=c.get(k)
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s doesn`t exist", k)
	}
	c.Set(k,v, d)
	c.mu.Unlock()
	return nil
}
//删除一个数据项
func (c *Cache) Delete(k string)  {
	c.mu.Lock()
	c.delete(k)
	c.mu.Unlock()
	
}
//将缓存数据写入io.write中
func (c *Cache) Save(w io.Writer)(err error)  {
	enc :=gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("使用Gob库注册项类型时出错")
		}
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, v :=range c.items{
		gob.Register(v.Object)
	}
	err = enc.Encode(&c.items)
	return
}

//保存数据项到文件里
func (c *Cache) SaveToFile(file string) error {
	f,err := os.Create(file)
	if err != nil {
		return err
	}
	return f.Close()

}

//从io。writer读取数据项
func (c *Cache) Load(r io.Reader) error{
	dec := gob.NewDecoder(r)
	items :=map[string]Item{}
	err :=dec.Decode(&items)
	if err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k,v := range items{
			ov, found := c.items[k]
			if !found || ov.Expired(){
				c.items[k] = v
			}

		}
	}
	return err

}
//从文件中加载缓存数据项
func (c *Cache) LoadFile(file string) error {
	f, err := os.Open(file)
	if err !=nil {
		return err
	}
	if err = c.Load(f); err !=nil{
		f.Close()
		return err
	}
	return f.Close()
}
// 返回缓存数据项的数量
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// 清空缓存
func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = map[string]Item{}
}

// 停止过期缓存清理
func (c *Cache) StopGc() {
	c.stopGc <- true
}

// 创建一个缓存系统
func NewCache(defaultExpiration, gcInterval time.Duration) *Cache {
	c := &Cache{
		defaultExpiration: defaultExpiration,
		gcInterval:        gcInterval,
		items:             map[string]Item{},
		stopGc:            make(chan bool),
	}
	// 开始启动过期清理 goroutine
	go c.gcLoop()
	return c
}

