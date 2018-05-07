package controller
import(
	"redisdb"
	"io/ioutil"
	"path"
	"sync"
	"os"
	"time"
	"strings"
	"strconv"
	gjson "github.com/tidwall/gjson"
	"myutils"
	"kube"
)
type StringSort []string

type ContainerType struct {
	Name string
	HashKey string
    Count int64  //该种容器在pipeline中总共需要多少个
    IdleQueue *myutils.StringQueue // 该种容器的空闲剩余队列
}
type ContainerPool struct {
	Map map[string]*ContainerType  // key = container name
	Rw *sync.RWMutex
}
type StepMap struct {
	Map map[string]*Step   // key = "step0,step1,..."
	Rw *sync.RWMutex
}
type SubStep struct {
	DeployId string   // deploy + hash(Sample + ContainerType + step + index)[0:6]
	StepStartTime time.Time
	StepRunTime  string
	CheckAlive bool
	LastAliveTime time.Time
	Status string
}
type Step struct {
    Command string  // 该step需要运行的命令
    Container string  // 该step需要运行容器的名称
    Status string  // 该step的状态，总共有四种类型: ready,running,succeed,failed
    Presteps []string  // 该step需要依赖的step
	SubSteps []*SubStep
}

type StepSimple struct {
    Command string              `json:command`
    Status string               `json:status`
    DeployId string             `json:deployId`
    StepRunTime  string         `json:stepRunTime`

}
type SendStepsInfo map[string]*StepSimple

type Controller struct {
	// sample 名称
	SampleName string
	// sample经过hash编码的sample前缀,这个前缀也是作为向客户端传递step信息的redis key
	Prefix string
	// 该sample对应的所有的step信息
	Steps *StepMap
	// 该sample相关的正在运行的deployment
	RunningDeployment  *myutils.Set 
	//关于此sample所涉及到的container
	DeploymentStatus *ContainerPool
	// 等待被调度运行的step
	WaitingRunningSteps *myutils.Set
	// 数据输出目录,为/mnt/data
	BaseDir string
	// sample开始执行的时间  
	StartTime time.Time 
	// 向step发送命令以后，会把信息存放到这个map中，等待deployment发回确认信息
	ConfirmSet *myutils.Set
	// sample的日志队列
	MessageQueue *myutils.StringQueue  // message format: deployId  + step + index + status
	LogsQueue   *myutils.StringQueue 
	// kubernetes配置信息
	Kube *kube.K8sClient
	// redis数据库配置信息
	Db redisdb.Database
	Status string
	// 发送信息到container的通道名称
	SendMessageToContainer string
	ListenContainerMessageChan string
	SendMessageToClient string
	ListenClient string
	Ticker5 *time.Ticker
	Ticker10 *time.Ticker
	Ticker15 *time.Ticker
	Ticker30 *time.Ticker
	Ticker60 *time.Ticker
	AliveTicker *time.Ticker
	Finished chan bool
	Failed chan bool
	Mu   *sync.Mutex
	DelConSignal  chan os.Signal
	RedisRunningSampleSet string
}
func NewController() (*Controller,error) {
	var reErr error
	sample := myutils.GetOsEnv("SAMPLE")
	jdata,err := ioutil.ReadFile(path.Join("/mnt/data",sample,"step0","pipeline.json"))
	if err != nil {
		return nil,err
	}	
	redisAddr := myutils.GetOsEnv("REDISADDR")
	sendToClient := myutils.GetOsEnv("SENDTOCLIENT")
	listenClient := myutils.GetOsEnv("LISTENCLIENT")
	prefix := myutils.GetSamplePrefix(sample)
	db := redisdb.NewRedisDB("tcp",redisAddr)
	k8s := kube.NewK8sClient(redisAddr)
	runningDeploy := myutils.NewSet()
	mytime := time.Now()
	ds := NewContainerPool()
	jsonObj := gjson.Parse(string(jdata))
	steps := NewStepMap()
	waitSteps := myutils.NewSet()
	confirm := myutils.NewSet()
	logsQueue := myutils.NewStringQueue(1000)
	jsonObj.ForEach(func(key,value gjson.Result) bool {
		if strings.Contains(key.String(),"step") {
			ikey := key.String()
			//waitSteps.Add(ikey)
			container := value.Get("Container").String()
			cmdCount := value.Get("CommandCount").Int()
			container = strings.Trim(container," ")
			conid := GetConHashKey(container)
			if !ds.Contain(conid) {
				val :=  &ContainerType{
					Count:1,
					Name: container,
					HashKey: conid,
					IdleQueue: myutils.NewStringQueue(2)}
				ds.Write(conid,val) 
			}else {
				ds.Read(conid).Count = ds.Read(conid).Count + cmdCount
			}
			command := value.Get("Command").String()
			commandName := value.Get("CommandName").String()
			args := value.Get("Args").String()
			cmd := command + " " + args
			cmdArray := make([]string,0,len(value.Get("SubArgs").Array()) + 1)
			subSteps := make([]*SubStep,0,len(value.Get("SubArgs").Array()) + 1)
			for ind,sub := range value.Get("SubArgs").Array() {
				cmdArray = append(cmdArray,cmd + " " + strings.Trim(sub.String()," "))
				subStep := &SubStep {
					StepStartTime: mytime,
					LastAliveTime: mytime,
					Status: "ready",
					DeployId: "not allocate",
					StepRunTime: "0h 0m 0s",
					CheckAlive: false}
				subSteps = append(subSteps,subStep)
				stepid := ikey + "-" + strconv.Itoa(ind)
				waitSteps.Add(stepid)
			}
			if len(cmdArray) == 0 {
				cmdArray = append(cmdArray,cmd)
				subStep := &SubStep {
                    StepStartTime: mytime,
                    LastAliveTime: mytime,
                    Status: "ready",
                    DeployId: "not allocate",
                    StepRunTime: "0h 0m 0s",
                    CheckAlive: false}
                subSteps = append(subSteps,subStep)
				stepid := ikey + "-" + "0"
				waitSteps.Add(stepid)
			}
			cmdStr := strings.Join(cmdArray,"-***-")
			cmdPath := path.Join("/mnt/data",sample,ikey,".command")
			err := ioutil.WriteFile(cmdPath,[]byte(cmdStr),0644)
			if err != nil {
				reErr = err
			}
			tpres := make([]string,0,len(value.Get("Prestep").Array()))
			for _,ps := range value.Get("Prestep").Array() {
				tpres = append(tpres,ps.String())
			}
			tdata := &Step{
				Command: commandName,
				Container: container,
				Presteps: tpres,
				Status: "ready",
				SubSteps: subSteps}
			steps.Write(ikey,tdata)
			return true
		}
		return true
	})
	if reErr != nil {
		return nil,reErr
	}
	return &Controller {
		SampleName: sample,
		Prefix: prefix,
		Steps: steps,
		Status: "",
		RedisRunningSampleSet: "Kubernetes-Running-Sample-Set",
		Mu: new(sync.Mutex),
		Ticker5: time.NewTicker(5 * time.Second),
		Ticker10: time.NewTicker(10 * time.Second),
		Ticker15: time.NewTicker(15 * time.Second),
		Ticker30: time.NewTicker(30 * time.Second),
		Ticker60: time.NewTicker(60 * time.Second),
		AliveTicker: time.NewTicker(30 * time.Second),
		RunningDeployment: runningDeploy,
		DeploymentStatus: ds,
		WaitingRunningSteps: waitSteps,
		BaseDir: "/mnt/data",
		StartTime: mytime,
		Db: db,
		Failed: make(chan bool),
		DelConSignal: make(chan os.Signal,1),
		ListenContainerMessageChan: prefix + "__" + "ReceiveFromCon",
		SendMessageToContainer: prefix + "__" + "SendToCon",
		ListenClient: listenClient,
		SendMessageToClient: sendToClient,
		Kube: k8s,
		Finished: make(chan bool),
		MessageQueue: myutils.NewStringQueue(1000),
		ConfirmSet: confirm,
		LogsQueue:logsQueue},nil

}
func IntToString(data int) string {
	return strconv.Itoa(data)
}
func StringToInt(data string) int {
	redata,err := strconv.Atoi(data)
	if err != nil {
		return -1
	}
	return redata

}
func GetConHashKey(name string) string {
	return "con" + myutils.GetSha256(name)[0:10]
}
func (s StringSort) Len() int {
	return len(s)
}
func (s StringSort) Swap(i,j int) {
	s[i],s[j] = s[j],s[i]
}
func (s StringSort) Less(i,j int) bool {
	left := strings.Split(strings.Split(s[i],"step")[1],"-")[0]
	right := strings.Split(strings.Split(s[j],"step")[1],"-")[0]
	lint,_ := strconv.ParseInt(left,10,32) 
	rint,_ := strconv.ParseInt(right,10,32)
	return lint < rint
}
func NewContainerPool() *ContainerPool {
	return &ContainerPool {
		Map: make(map[string]*ContainerType),
		Rw: new(sync.RWMutex)}
}
func (cp *ContainerPool) Read(key string) *ContainerType {
	cp.Rw.RLock()
	defer cp.Rw.RUnlock()
	if _,ok := cp.Map[key]; ok {
		return cp.Map[key]
	}else {
		return nil
	}

}
func (cp *ContainerPool) Write(key string,value *ContainerType) {
	cp.Rw.Lock()
	defer cp.Rw.Unlock()
	cp.Map[key] = value
}
func (cp *ContainerPool) Contain(key string) bool {
	cp.Rw.RLock()
	defer cp.Rw.RUnlock()
	if _,ok := cp.Map[key];ok {
		return true
	}
	return false

}
func  NewStepMap() *StepMap {
	return &StepMap{Map:make(map[string]*Step),Rw: new(sync.RWMutex)}
}  
func (sp *StepMap) Read(key string) *Step {
	sp.Rw.RLock()
	defer sp.Rw.RUnlock()
	if _,ok := sp.Map[key];ok {
		return sp.Map[key]
	}
	return nil
}
func (sp *StepMap) Write(key string,value *Step) {
	sp.Rw.Lock()
	defer sp.Rw.Unlock()
	sp.Map[key] = value
}
func (sp *StepMap) Members() map[string]*Step {
	sp.Rw.RLock()
	defer sp.Rw.RUnlock()
	return sp.Map
}
func (sp *StepMap) Contains(key string) bool {
	sp.Rw.RLock()
	defer sp.Rw.RUnlock()
	if _,ok := sp.Map[key];ok {
		return true
	}
	return false
}
func (sp *StepMap) Len() int {
	sp.Rw.RLock()
	defer sp.Rw.RUnlock()
	return len(sp.Map)
}
