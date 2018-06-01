package controller
import(
	"redisdb"
	"io/ioutil"
	"path"
	"sync"
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
	ResourcesLimits []string
	ResourcesRequests []string
}

type StepSimple struct {
    Command string              `json:command`
    Status string               `json:status`
    DeployId string             `json:deployId`
    StepRunTime  string         `json:stepRunTime`

}
type SendStepsInfo map[string]*StepSimple

type Job struct {
	// sample 名称
	RecoveryMap *myutils.Set
	SampleName string
	// sample经过hash编码的sample前缀,这个前缀也是作为向客户端传递step信息的redis key
	Prefix string
	// 该sample对应的所有的step信息
	Steps *StepMap
	// 该sample相关的正在运行的deployment
	RunningDeployment  *myutils.Set 
	// 等待被调度运行的step
	WaitingRunningSteps *myutils.Set
	// 数据输出目录,为/mnt/data
	BaseDir string
	// sample开始执行的时间  
	StartTime time.Time 
	// 向step发送命令以后，会把信息存放到这个map中，等待deployment发回确认信息
	LogsQueue   *myutils.StringQueue 
	// kubernetes配置信息
	Kube *kube.Kube
	// redis数据库配置信息
	Db redisdb.Database
	Status string
	FinishSignal chan <- string
	StepStatus string
	TemplateEstimate string
	TemplateName string
	Mu *sync.Mutex
	DeleteLocker *sync.Mutex
	WriteStepStatus   bool
}
func NewJob(redisAddr,sample string,fchan chan <- string,init *kube.InitArgs,del *sync.Mutex,recovery *myutils.Set) (*Job,error) {
	var reErr error
	jdata,err := ioutil.ReadFile(path.Join("/mnt/data",sample,"step0","pipeline.json"))
	if err != nil {
		return nil,err
	}	
	prefix := myutils.GetSamplePrefix(sample)
	db := redisdb.NewRedisDB("tcp",redisAddr)
	k8s := kube.NewKube(init)
	runningDeploy := myutils.NewSet()
	mytime := time.Now()
	jsonObj := gjson.Parse(string(jdata))
	steps := NewStepMap()
	waitSteps := myutils.NewSet()
	logsQueue := myutils.NewStringQueue(1000)
	jsonObj.ForEach(func(key,value gjson.Result) bool {
		if strings.Contains(key.String(),"step") {
			ikey := key.String()
			//waitSteps.Add(ikey)
			container := value.Get("Container").String()
			container = strings.Trim(container," ")
			command := value.Get("Command").String()
			commandName := value.Get("CommandName").String()
			args := value.Get("Args").String()
			cmd := command + " " + args
			limits := make([]string,2,2)
			requests := make([]string,2,2)
			cmdArray := make([]string,0,len(value.Get("SubArgs").Array()) + 1)
			subSteps := make([]*SubStep,0,len(value.Get("SubArgs").Array()) + 1)
			for i,t := range value.Get("ResourcesLimits").Array() {
				limits[i] = t.String()
			}
			for i,t := range value.Get("ResourcesRequests").Array() {
				requests[i] = t.String()
			}
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
				ResourcesRequests: requests,
				ResourcesLimits: limits,
				SubSteps: subSteps}
			steps.Write(ikey,tdata)
			return true
		}
		return true
	})
	if reErr != nil {
		return nil,reErr
	}
	return &Job {
		SampleName: sample,
		Prefix: prefix,
		Steps: steps,
		Status: "",
		WriteStepStatus: true,
		DeleteLocker: del,
		Mu: new(sync.Mutex),
		StepStatus: "",
		RecoveryMap: recovery,
		RunningDeployment: runningDeploy,
		WaitingRunningSteps: waitSteps,
		BaseDir: "/mnt/data",
		StartTime: mytime,
		Db: db,
		Kube: k8s,
		FinishSignal: fchan,
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
