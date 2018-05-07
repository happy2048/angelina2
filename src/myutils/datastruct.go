package myutils
import(
	"sync"
	"time"
)

/*
 	create a Set 

*/
type Set struct {
	Map map[string]int64
	Rw *sync.RWMutex
} 

type StringQueue struct{
    Mu *sync.Mutex
    Queue []string
    Cap  int
}
type Dict struct {
	Map map[string]string
	Rw  *sync.RWMutex
}
type SortSet struct {
	Map map[string]bool
	Queue []string
	Mu  *sync.Mutex
	Cap int
}
func NewSortSet(length int) *SortSet {
	return &SortSet{
		Queue: make([]string,0,length),
		Map: make(map[string]bool),
		Cap: length,
		Mu: new(sync.Mutex)}
}
func (ss *SortSet) Push(data string) {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	if len(ss.Queue) + 1 >= ss.Cap {
		return 
	}
	if _,ok := ss.Map[data];!ok {
		ss.Map[data] = true
		ss.Queue = append(ss.Queue,data)
	}

}
func (ss *SortSet) Pop() string {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	var redata string
	if len(ss.Queue) == 0 {
		return ""
	}else if len(ss.Queue) == 1 {
        redata = ss.Queue[0]
        ss.Queue = ss.Queue[0:0]
    }else {
        redata = ss.Queue[0]
        ss.Queue = ss.Queue[1:]
    }
	delete(ss.Map,redata)   
    return redata
}
func (ss *SortSet) PopAll() []string {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	redata := ss.Queue
	ss.Queue = ss.Queue[0:0]
	for _,val := range redata {
		delete(ss.Map,val)
	}
	return redata
}
func (ss *SortSet) Len() int {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	return len(ss.Queue)
}
func (ss *SortSet) Contain(data string) bool {
	ss.Mu.Lock()
	defer ss.Mu.Unlock()
	if _,ok := ss.Map[data];ok {
		return true
	}
	return false
}	
func NewDict() *Dict {
	return &Dict {
		Map: make(map[string]string),
		Rw: new(sync.RWMutex)} 
}
func (dict *Dict) SetValue(key,value string) {
	dict.Rw.Lock()
	defer dict.Rw.Unlock()
	dict.Map[key] = value

}
func (dict *Dict) DeleteValue(key string) {
	dict.Rw.Lock()
	defer dict.Rw.Unlock()
	delete(dict.Map,key)
}
func (dict *Dict) ReadValue(key string) string {
	dict.Rw.RLock()
	defer dict.Rw.RUnlock()
	redata := ""
	if _,ok := dict.Map[key]; ok {
		redata = dict.Map[key]

	}
	return redata
}
func (dict *Dict) Len() int {
	dict.Rw.RLock()
	defer dict.Rw.RUnlock()
	return len(dict.Map)

}
func (dict *Dict) Members() map[string]string {
	dict.Rw.RLock()
	defer dict.Rw.RUnlock()
	return dict.Map
}


func  NewSet() *Set {
    return &Set{
		Map: make(map[string]int64),
		Rw: new(sync.RWMutex)}
}
func (set *Set) Add(item string) bool {
	set.Rw.Lock()
	defer set.Rw.Unlock()
    if _,ok := set.Map[item]; !ok {
        set.Map[item] = time.Now().Unix()
        return true
    }else {
        return false
    }
}
func (set *Set) Remove(item string) {
	set.Rw.Lock()
	defer set.Rw.Unlock()
    delete(set.Map,item)
}
func (set *Set) Contains(item string) bool {
	set.Rw.RLock()
	defer set.Rw.RUnlock()
    if _,ok := set.Map[item]; !ok {
        return false
    }else {
        return true
    }
}
func (set *Set) Members() []string {
	set.Rw.RLock()
	defer set.Rw.RUnlock()
    reArr := make([]string,0,len(set.Map))
    for key,_ := range set.Map {
        reArr = append(reArr,key)
    }
    return reArr

}
func (set *Set) Len() int {
    set.Rw.RLock()
	defer set.Rw.RUnlock()
	return len(set.Map)
}
func (set *Set) Timestamp(key string) int64 {
	set.Rw.RLock()
	defer set.Rw.RUnlock()
	if _,ok := set.Map[key];ok {
		return set.Map[key]
	}
	return -1
}

func NewStringQueue(qlen int) *StringQueue {
	return &StringQueue{
		Mu: new(sync.Mutex),
		Queue: make([]string,0,qlen),
		Cap: qlen}
}
func (sq *StringQueue) PopFromQueue() string {
	sq.Mu.Lock()
	defer sq.Mu.Unlock()
	var redata string
	if len(sq.Queue) == 0 {
		redata = ""
	}else if len(sq.Queue) == 1 {
		redata = sq.Queue[0]
		sq.Queue = sq.Queue[0:0]
	}else {
		redata = sq.Queue[0]
		sq.Queue = sq.Queue[1:]
	}
	return redata

} 
func (sq *StringQueue) Len() int {
	sq.Mu.Lock()
	defer sq.Mu.Unlock()
	return 	len(sq.Queue)

}
func (sq *StringQueue) PopAllFromQueue() []string {
	sq.Mu.Lock()
    defer sq.Mu.Unlock()
	data := sq.Queue
	sq.Queue = sq.Queue[0:0]
	return data

}
func (sq *StringQueue) PushToQueue(data string) bool {
	sq.Mu.Lock()
	defer sq.Mu.Unlock()
	if len(sq.Queue)  + 1 >= sq.Cap {
		return false
	}else {
		sq.Queue = append(sq.Queue,data)
		return true
	}
} 

