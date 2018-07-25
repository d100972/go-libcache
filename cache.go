package cache

/*****************************************************************************************
 * Golang 实现 缓存组件
 *
 * 系统环境：Deepin Linux 15.6 x64/GO 1.10.2
 * 文件名称：cache.go
 * 内容摘要：实现类似memcache、redis的缓存系统。
 * 其他说明：当前基础功能包括：缓存数据的存储、过期数据项的管理、内存数据导入导出、提供CRUD接口。
 * 当前版本：1.0
 * 作    者：邓小佳
 * 完成时期：2018.07.25
 *
 * 修改记录1：
 * 修改日期 ：
 * 版 本 号 ：
 * 修 改 人 ：
 * 修改内容 ：
 *
 ****************************************************************************************/
// 包
import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

/***************************************************************************************/
// 数据结构与常量

type Item struct { // 缓存中存储的数据项结构
	Object     interface{} // 缓存中存储的数据项
	Expiration int64       // 该数据项生存的时间
}

type Cache struct { // 缓存系统结构
	defaultExpiration time.Duration   // 数据项是否会过期标志
	items             map[string]Item // 用于存储缓存数据项
	mux               sync.RWMutex    // 读写锁
	gcInterval        time.Duration   // 过期数据项清理周期
	stopGc            chan bool       // 是否停止缓存回收清理
}

const (
	NoExpiration      time.Duration = -1 // 永不过期的标志
	DefaultExpiration time.Duration = 0  // 有默认过期时间的标志
)

/***************************************************************************************/

/***************************************************************************************
 * 功能描述：创建一个缓存
 * 输入参数：是否会过期标志：defaultExpiration, 过期周期标志：gcInterval
 * 输出参数：无
 * 返 回 值：一个新的缓存
 * 其他说明：无
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func NewCache(defaultExpiration, gcInterval time.Duration) *Cache {
	newCache := &Cache{
		defaultExpiration: defaultExpiration,
		gcInterval:        gcInterval,
		items:             map[string]Item{},
	}
	go newCache.gcLoop() // 启动缓存项过期回收清理 goroutine
	return newCache
}

/***************************************************************************************
 * 功能描述：判断数据项是否已经过期
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：过期为true
 * 其他说明：该函数为 Item 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisItem Item) Expired() bool {
	if thisItem.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > thisItem.Expiration // 使用Unix时间戳，单位纳秒，若当前时间大于过期时间，则判断为过期
}

/***************************************************************************************
 * 功能描述：过期缓存数据项回收清理
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) gcLoop() {
	ticker := time.NewTicker(thisCache.gcInterval) // 创建一个ticker时钟，通过指定的参数:gcInterval时间间隔
	// 周期性的从ticker.C管道中发送数据过来。
	for {
		select {
		case <-ticker.C:
			thisCache.DeleteExpired() // 周期性的执行删除过期缓存数据项
		case <-thisCache.stopGc: // 为保证gcLoop能正常结束，监听stopGc管道
			ticker.Stop()
			return
		}
	}
}

/***************************************************************************************
 * 功能描述：通过键值删除缓存数据项
 * 输入参数：键值 key string
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) delete(key string) {
	delete(thisCache.items, key)
}

/***************************************************************************************
 * 功能描述：通过键名删除一个数据项，可导出
 * 输入参数：数据项键名：key string
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Delete(key string) {
	thisCache.mux.Lock()
	thisCache.delete(key)
	thisCache.mux.Unlock()
}

/***************************************************************************************
 * 功能描述：删除过期的缓存数据项
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) DeleteExpired() {
	now := time.Now().UnixNano()
	thisCache.mux.Lock()
	defer thisCache.mux.Unlock()

	for key, val := range thisCache.items { // 遍历所有数据项，删除过期数据项
		if val.Expiration > 0 && now > val.Expiration {
			thisCache.delete(key)
		}
	}
}

/***************************************************************************************
 * 功能描述：设置缓存数据项，若数据项存在则覆盖，无锁操作
 * 输入参数：数据项键名：key string, 数据项键值：value interface{}, 数据项生命周期：dur time.Duration
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) set(key string, value interface{}, dur time.Duration) {
	var expir int64
	if dur == DefaultExpiration {
		dur = thisCache.defaultExpiration
	}
	if dur > 0 {
		expir = time.Now().Add(dur).UnixNano()
	}

	thisCache.items[key] = Item{
		Object:     value,
		Expiration: expir,
	}
}

/***************************************************************************************
 * 功能描述：设置缓存数据项，若数据项存在则覆盖,导出函数
 * 输入参数：数据项键名：key string, 数据项键值：value interface{}, 数据项生命周期：dur time.Duration
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Set(key string, value interface{}, dur time.Duration) {
	var expir int64
	if dur == DefaultExpiration {
		dur = thisCache.defaultExpiration
	}
	if dur > 0 {
		expir = time.Now().Add(dur).UnixNano()
	}
	thisCache.mux.Lock()
	defer thisCache.mux.Unlock()

	thisCache.items[key] = Item{
		Object:     value,
		Expiration: expir,
	}
}

/***************************************************************************************
 * 功能描述：获取数据项，若找到数据项，还需要判断数据项是否已经过期，无锁
 * 输入参数：数据项键名：key string
 * 输出参数：无
 * 返 回 值：具体数据项的值以及是否找到(bool)
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) get(key string) (interface{}, bool) {
	item, found := thisCache.items[key]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}
	return item.Object, found
}

/***************************************************************************************
 * 功能描述：获取数据项，若找到数据项，还需要判断数据项是否已经过期
 * 输入参数：数据项键名：key string
 * 输出参数：无
 * 返 回 值：具体数据项的值以及是否找到(bool)
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Get(key string) (interface{}, bool) {
	thisCache.mux.RLock()
	defer thisCache.mux.RUnlock()

	item, found := thisCache.items[key]
	if !found {
		return nil, false
	}
	if item.Expired() {
		return nil, false
	}
	return item.Object, true
}

/***************************************************************************************
 * 功能描述：添加数据项，若已存在，返回错误
 * 输入参数：数据项键名：key string, 数据项键值：value interface{}, 数据项生命周期：dur time.Duration
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180724      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Add(key string, val interface{}, dur time.Duration) error {
	thisCache.mux.Lock()
	_, found := thisCache.get(key)
	if found {
		thisCache.mux.Unlock()
		return fmt.Errorf("Item %s already exists", key)
	}
	thisCache.set(key, val, dur)
	thisCache.mux.Unlock()
	return nil
}

/***************************************************************************************
 * 功能描述：替换一个存在的数据项
 * 输入参数：数据项键名：key string, 数据项键值：value interface{}, 数据项生命周期：dur time.Duration
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Replace(key string, val interface{}, dur time.Duration) error {
	thisCache.mux.Lock()
	_, found := thisCache.get(key)
	if !found {
		thisCache.mux.Unlock()
		return fmt.Errorf("item doesn't exist.", key)
	}
	thisCache.set(key, val, dur)
	thisCache.mux.Unlock()
	return nil
}

/***************************************************************************************
 * 功能描述：将缓存数据项写入到io.Writer中
 * 输入参数：wrt io.Writer
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Save(wrt io.Writer) (err error) {
	encode := gob.NewEncoder(wrt) // 序列化操作，进行编码操作
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Error registring item type with Gob lib.")
			return
		}
	}()

	thisCache.mux.RLock()
	defer thisCache.mux.RUnlock()

	for _, val := range thisCache.items {
		gob.Register(val.Object) // 因为item的值为interface{}，所以需要注册
	}
	err = encode.Encode(&thisCache.items)
	return
}

/***************************************************************************************
 * 功能描述：将缓存数据项从内存中保存到文件中
 * 输入参数：file string 要打开的文件名
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) SaveMemToFile(file string) error {
	fp, err := os.Create(file)
	if err != nil {
		return err
	}
	if err = thisCache.Save(fp); err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

/***************************************************************************************
 * 功能描述：从io.Reader中读取数据项
 * 输入参数：rd io.Reader
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Load(rd io.Reader) error {
	decode := gob.NewDecoder(rd) // 解码，反序列化
	items := map[string]Item{}
	err := decode.Decode(&items)
	if err != nil {
		return err
	}
	thisCache.mux.Lock()
	defer thisCache.mux.Unlock()

	for key, val := range items {
		theItem, found := thisCache.items[key]
		if found && !theItem.Expired() {
			thisCache.items[key] = val
		}
	}
	return nil
}

/***************************************************************************************
 * 功能描述：将缓存数据项从文件中恢复加载到内存中
 * 输入参数：file string 要打开的文件名
 * 输出参数：无
 * 返 回 值：无 error， 则为 nil
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) LoadFileToMem(file string) error {
	fp, err := os.Open(file)
	if err != nil {
		return err
	}
	if err = thisCache.Load(fp); err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

/***************************************************************************************
 * 功能描述：统计当前缓存数据项的数量
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：int 缓存数量
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Count() int {
	thisCache.mux.RLock()
	defer thisCache.mux.RUnlock()
	return len(thisCache.items)
}

/***************************************************************************************
 * 功能描述：清空缓存
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) Flush() {
	thisCache.mux.Lock()
	defer thisCache.mux.Unlock()
	thisCache.items = map[string]Item{}
}

/***************************************************************************************
 * 功能描述：停止过期缓存清理方法gcLoop()
 * 输入参数：无
 * 输出参数：无
 * 返 回 值：无
 * 其他说明：该函数为 Cache 类方法
 *
 * 修改日期      版本号      修改人      修改内容
 * ------------------------------------------------------------------------------------
 * 20180725      v1.0        邓小佳      创建
 * ************************************************************************************/
func (thisCache *Cache) StopGc() {
	thisCache.stopGc <- true
}
