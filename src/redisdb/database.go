package redisdb
import(
	"fmt"
	"time"
	"strconv"
	//"reflect"
	redis "github.com/garyburd/redigo/redis"
)
/*
func main() {
	rdb := NewRedisDB("tcp","127.0.0.1:6379")
	var db Database
	db = rdb
	status,err := db.RedisStringSet("mytest","ok")
	fmt.Println(status,err)
	redata,err := db.RedisStringGet("mytest")
	fmt.Println(redata,err)
	isExist,err := db.RedisKeyExist("mytest")
	time.Sleep(5 * time.Second)
	fmt.Println(isExist,err)
	db.RedisHashSet("test1","b","34")
	status1,err := db.RedisHashGet("test1","ab")
	fmt.Println(status1,err)
	db.RedisListRpush("list1","okok")
	db.RedisListLrange("list1",0,8)
	//db.RedisListLpop("list1")
	fmt.Println("len:",db.RedisListLlen("list1"))
	db.RedisListValueOfIndex("list1",6)
	//db.RedisDelKey("list1")
	db.RedisSetAdd("aaa","bb")
	db.RedisSetAdd("aaa","cc")
	db.RedisSetMembers("aaa")
	db.RedisPublish("redChatRoom","23")
	//db.RedisSubscribe("redChatRoom",func(data string){fmt.Printf("Message: %s\n",data)})
}
*/
type Database interface {
	RedisStringSet(key,value string) (string,error)
	RedisStringGet(key string) (string,error)
	RedisHashSet(key,field,value  string) (bool,error)
	RedisHashGet(key,field string) (string,error)
	RedisHashIncry(key,field string,number int) (bool,error)
	//RedisHashMget(key string,fields []string) ([]string,error)
	//RedisHashMset(key string,[][]string) (bool,error)
	//RedisHashHdel(Key string,fields []string) (bool,error)
	//RedisHashHexists(key string,field string) (bool,error)
	//RedisHashHvals(key string,fields []string) ([]string,error)
	RedisHashHkeys(key string) ([]string,error)
	RedisListRpush(key string,value string) (int,error)
	RedisListLrange(key string,start,end int) ([]string,error)
	RedisListLpop(key string) (string,error)
	RedisListLlen(key string) (int,error)
	RedisListValueOfIndex(key string,index int) (string,error)
	RedisDelKey(key string) (bool,error)
	RedisStringSetWithEx(key,value string,ex int) (string,error)
	RedisKeyExist(key string) (bool,error)
	RedisGetKeys(patten string) ([]string,error)
	RedisSetAdd(key,value string) (bool,error)
	RedisSetMembers(key string) ([]string,error)
	RedisSetSremMember(key,member string) (bool,error)
	RedisSetSisMember(key,member string) (bool,error)
	RedisSetMemberCount(key string) (int,error)
	RedisSubscribe(channel string,transferStatus func(string))
	RedisPublish(channel,data string) (bool,error)
	
}
type RedisDB struct {
	protocol string
	addr string
	pool *redis.Pool
	
}
func NewRedisDB(protocol,addr string) (*RedisDB) {
	return &RedisDB {
		protocol: protocol,
		addr: addr,
		pool: createRedisPool(protocol,addr)}

}
func createRedisPool(protocol,addr string) (*redis.Pool) {
	return &redis.Pool{
		MaxIdle: 3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn,error) {return redis.Dial(protocol,addr)}}
}
func RedisTestConnect(addr string) (error) {
	c,err := redis.Dial("tcp",addr)
	if err != nil {
		return err
	}
	defer c.Close()
	return nil
}
func (rdb *RedisDB) RedisSetMemberCount(key string) (int,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	count,err := redis.Int(conn.Do("SCARD",key))
	return count,err
}
func (rdb *RedisDB) RedisSetSremMember(key,member string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Bool(conn.Do("SREM",key,member))
	return status,err
}
func (rdb *RedisDB) RedisHashHkeys(key string) ([]string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Strings(conn.Do("HKEYS",key))
	return redata,err

}
func (rdb *RedisDB) RedisHashIncry(key,field string,number int) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Bool(conn.Do("HINCRBY",key,field,number))
	return status,err
}
func (rdb *RedisDB) RedisPublish(channel,data string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Bool(conn.Do("PUBLISH",channel,data))
	return status,err
}
func (rdb *RedisDB) RedisGetKeys(patten string) ([]string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	keys,err := redis.Strings(conn.Do("KEYS",patten))
	return keys,err
}
func (rdb *RedisDB) RedisSubscribe(channel string,transferStatus func(string)) {
	conn := rdb.pool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{conn}
	psc.Subscribe(channel)
	for {
		switch v := psc.Receive().(type) {
			case redis.Message:
				transferStatus(string(v.Data))
			case error:
				fmt.Println(v)
				return
		}
	}

}
func (rdb *RedisDB) RedisSetMembers(key string) ([]string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Strings(conn.Do("SMEMBERS",key))
	return redata,err
}
func (rdb *RedisDB) RedisSetSisMember(key,member string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Bool(conn.Do("SISMEMBER",key,member))
	return redata,err
}
func (rdb *RedisDB) RedisSetAdd(key,value string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Bool(conn.Do("SADD",key,value))
	return status,err

}
func (rdb *RedisDB) RedisDelKey(key string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Bool(conn.Do("DEL",key))
	return status,err
} 
func (rdb *RedisDB) RedisListValueOfIndex(key string,index int) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.String(conn.Do("LINDEX",key,index))
	return redata,err
}
func (rdb *RedisDB) RedisListLlen(key string) (int,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Int(conn.Do("LLEN",key))
	return redata,err
}
func (rdb *RedisDB) RedisListRpush(key string,value string) (int,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.Int(conn.Do("RPUSH",key,value))
	return status,err

}
func (rdb *RedisDB) RedisListLrange(key string,start,end int) ([]string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Strings(conn.Do("LRANGE",key,start,end))
	fmt.Println(redata)
	return redata,err
}
func (rdb *RedisDB) RedisListLpop(key string) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.String(conn.Do("LPOP",key))
	return redata,err
}
func (rdb *RedisDB) RedisStringSet(key,value string) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.String(conn.Do("SET",key,value))
	return status,err
}
func (rdb *RedisDB) RedisStringSetWithEx(key,value string,ex int) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	status,err := redis.String(conn.Do("SET",key,value,"EX", strconv.Itoa(ex)))
	return status,err
}
func (rdb *RedisDB) RedisStringGet(key string) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.String(conn.Do("GET",key))
	return redata,err

}
func (rdb *RedisDB) RedisKeyExist(key string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	isExist,err := redis.Bool(conn.Do("EXISTS",key))
	return isExist,err
}
func (rdb *RedisDB) RedisHashSet(key,field,value string) (bool,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.Bool(conn.Do("HSET",key,field,value))
	return redata,err

}
func (rdb *RedisDB) RedisHashGet(key,field string) (string,error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	redata,err := redis.String(conn.Do("HGET",key,field))
	return redata,err
}




	
