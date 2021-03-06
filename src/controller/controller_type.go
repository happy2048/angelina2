package controller
import(
	"kube"
	"myutils"
	"redisdb"
	"strings"
	"strconv"
	"sync"
	"time"
	"io/ioutil"
)
type SendEmailMessage struct {
    To    []string
    SmtpServer string
    Port   int
    User   string
    Passwd string
}
type SimpleJob struct {
	Name string
	FinishedTime time.Time
	Status string
	Log  string
}
type FinishJobsMap struct {
	Mu *sync.Mutex
	Map map[string]*SimpleJob
}
type JobsMap struct {
	Rw *sync.RWMutex
	Map map[string]*Job
}
type JobsNameMap struct {
	Mu *sync.Mutex
	Map map[string]string
}
type Controller struct {
	WaitingRunJobs     *myutils.Set
	CancelJobs         *myutils.Set
	StartingJobs     *myutils.Set
	RunningJobs        *myutils.Set
	WaitingDeleteJobs  *myutils.Set
	DeletingJobs       *myutils.Set
	SendMailJobs       *myutils.Set    
	FinishedJobs       *FinishJobsMap
	JobsPool		   *JobsMap
	MessageQueue       *myutils.StringQueue
	LogsQueue          *myutils.StringQueue
	Kube 			   *kube.Kube
	Db 				   redisdb.Database
	Ticker5			   *time.Ticker
	LogTicker		   *time.Ticker
	WriteLogTicker     *time.Ticker
	Ticker20		   *time.Ticker
	Ticker30		   *time.Ticker
	Ticker60		   *time.Ticker
	Ticker10		   *time.Ticker
	HandleDataTicker   *time.Ticker
	Service            string
	RedisAddr          string
	FinishedSignal     chan string
	BackupKey		   string
	NameMap            *JobsNameMap     
	RunnerCmdPath      string
	KubeConfig        *kube.InitArgs
	DeleteLocker      *sync.Mutex
	Recovery           bool
	RecoveryMap       *myutils.Set
	SendEmailInfo     SendEmailMessage
	SmtpEnabled       bool
	SendEmailTicker   *time.Ticker
}

func NewController() *Controller {
	redis1 := myutils.GetOsEnv("ANGELINA_REDIS_ADDR")
	redisPort := myutils.GetOsEnv("ANGELINA_REDIS_PORT")
	redisAddr := redis1 + ":" + redisPort
	angelina := myutils.GetOsEnv("ANGELINA_SERVER")
	db := redisdb.NewRedisDB("tcp",redisAddr)
	rdata,err := ioutil.ReadFile("/root/angelina-runner-pod.yml")
	if err != nil {
		myutils.Print("Error","read deployment template file failed,reason:" + err.Error(),true)
	}
	smtpOk := myutils.GetOsEnv("SMTP_ENABLED")
	var smtpSend bool
	if strings.Trim(smtpOk," ") != "true" {
		smtpSend = false
	}else {
		smtpSend = true
	}
	smtpServer := myutils.GetOsEnv("SMTP_SERVER")
	smtpServerPort := myutils.GetOsEnv("SMTP_SERVER_PORT")
	smtpEmailTo  := myutils.GetOsEnv("EMAIL_TO")
	smtpUser := myutils.GetOsEnv("EMAIL_SMTP_USER")
	smtpPass := myutils.GetOsEnv("EMAIL_SMTP_PASS")
	if smtpServer == "" || smtpServerPort == "" || smtpEmailTo == "" || smtpUser == "" || smtpPass == "" {
		smtpSend = false
	}
	p,err := strconv.Atoi(strings.Trim(smtpServerPort," "))
	if err != nil {
		smtpSend = false
		p = 0
	}
	
	emailInfo := SendEmailMessage {To:strings.Split(smtpEmailTo,","),SmtpServer: strings.Trim(smtpServer," "),Port: p,User: strings.Trim(smtpUser," "),Passwd: strings.Trim(smtpPass," ")} 
	sendEmailInt := myutils.GetOsEnv("EMAIL_SEND_INTERVAL")
	sendEmailInter,err := strconv.Atoi(strings.Trim(sendEmailInt," "))
	if err != nil {
		sendEmailInter = 30
	}
	init := &kube.InitArgs {
		ApiServer: myutils.GetOsEnv("KUBER_APISERVER"),
		ControllerEntry: myutils.GetOsEnv("ANGELINA_CONTROLLER_ENTRY"),
		Namespace: myutils.GetOsEnv("NAMESPACE"),
		StartCmd: myutils.GetOsEnv("START_CMD"),
		DeploymentTemp: string(rdata),
		RedisAddr: redisAddr,
		QuotaName: myutils.GetOsEnv("ANGELINA_QUOTA"),
		GlusterfsEndpoint: myutils.GetOsEnv("GLUSTERFS_ENDPOINT"),
		GlusterfsDataVolume: myutils.GetOsEnv("GLUSTERFS_DATA_VOLUME"),
		GlusterfsReferVolume: myutils.GetOsEnv("GLUSTERFS_REFER_VOLUME")}
	k8s := kube.NewKube(init)
	contr:=  &Controller {
		Kube: k8s,
		Db: db,
		SmtpEnabled: smtpSend,
		KubeConfig: init,
		Recovery: true,
		RecoveryMap: myutils.NewSet(),
		Ticker5: time.NewTicker(5 * time.Second),
		Ticker10: time.NewTicker(10 * time.Second),
		Ticker20: time.NewTicker(20 * time.Second),
		Ticker30: time.NewTicker(30 * time.Second),
		Ticker60: time.NewTicker(60 * time.Second),
		LogTicker: time.NewTicker(10 * time.Second),
		WriteLogTicker: time.NewTicker(30 * time.Second),
		HandleDataTicker: time.NewTicker(30 * time.Second),
		WaitingRunJobs: myutils.NewSet(),
		StartingJobs: myutils.NewSet(),
		CancelJobs: myutils.NewSet(),
		Service: angelina,
		NameMap: NewJobsNameMap(),
		JobsPool:NewJobsMap(),
		RunnerCmdPath:"/root/angelina-runner",
		MessageQueue: myutils.NewStringQueue(1000),
		LogsQueue: myutils.NewStringQueue(1000),
		RunningJobs: myutils.NewSet(),
		FinishedJobs: NewFinishJobsMap(),
		WaitingDeleteJobs: myutils.NewSet(),
		RedisAddr: redisAddr,
		DeleteLocker: new(sync.Mutex),
		BackupKey: "angelina-running-jobs",
		FinishedSignal: make(chan string),
		DeletingJobs: myutils.NewSet()}
	if smtpSend == true {
		contr.SendEmailTicker = time.NewTicker(time.Duration(sendEmailInter * 60) * time.Second)
		contr.SendEmailInfo = emailInfo
		contr.SendMailJobs = myutils.NewSet()
	}
	return contr
}
func NewJobsMap() *JobsMap {
	return &JobsMap {
		Rw: new(sync.RWMutex),
		Map: make(map[string]*Job)}
}

func (jm *JobsMap) Read(key string) *Job {
	jm.Rw.RLock()
	defer jm.Rw.RUnlock()
	if _,ok := jm.Map[key];ok {
		return jm.Map[key]
	}
	return nil
}
func (jm *JobsMap) Write(key string,value *Job) {
	jm.Rw.Lock()
	defer jm.Rw.Unlock()
	jm.Map[key] = value

}
func (jm *JobsMap) Contain(key string) bool {
	jm.Rw.RLock()
	defer jm.Rw.RUnlock()
	if _,ok := jm.Map[key];ok {
		return true
	}
	return false
}
func (jm *JobsMap) Members() map[string]*Job {
	jm.Rw.RLock()
	defer jm.Rw.RUnlock()
	return jm.Map

}
func (jm *JobsMap) Len() int {
	jm.Rw.RLock()
	defer jm.Rw.RUnlock()
	return len(jm.Map)
}

func (jm *JobsMap) Delete(key string) {
	jm.Rw.Lock()
	defer jm.Rw.Unlock()
	delete(jm.Map,key)
}
func (fjm *FinishJobsMap) Add(key string,value *SimpleJob) {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	fjm.Map[key] = value
}
func (fjm *FinishJobsMap) Members() map[string]*SimpleJob {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	return fjm.Map
}
func (fjm *FinishJobsMap) Remove(key string) {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	delete(fjm.Map,key)
}
func (fjm *FinishJobsMap) Contain(key string) bool {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	if _,ok := fjm.Map[key];ok {
		return true
	}
	return false

}
func (fjm *FinishJobsMap) Names() []string {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	redata := make([]string,0,len(fjm.Map))
	for _,data := range fjm.Map {
		redata = append(redata,data.Name)
	}
	return redata

}
func (fjm *FinishJobsMap) Read(key string) *SimpleJob {
	fjm.Mu.Lock()
	defer fjm.Mu.Unlock()
	if _,ok := fjm.Map[key];ok {
		return fjm.Map[key]
	}
	return nil
}
func NewFinishJobsMap() *FinishJobsMap {
	return &FinishJobsMap{
		Mu: new(sync.Mutex),
		Map: make(map[string]*SimpleJob)}
}
func NewJobsNameMap() *JobsNameMap {
	return &JobsNameMap{
		Mu: new(sync.Mutex),
		Map: make(map[string]string)}
}
func (jnm *JobsNameMap) Add(key,value string) {
	jnm.Mu.Lock()
	defer jnm.Mu.Unlock()
	jnm.Map[key] = value

}
func (jnm *JobsNameMap) Read(key string) string {
	jnm.Mu.Lock()
	defer jnm.Mu.Unlock()
	if _,ok := jnm.Map[key];ok {
		return jnm.Map[key]
	}
	return ""
}
func (jnm *JobsNameMap) Contain(key string) bool {
	jnm.Mu.Lock()
	defer jnm.Mu.Unlock()
	if _,ok := jnm.Map[key];ok {
		return true
	}
	return false
}	

func (jnm *JobsNameMap) Remove(key string) {
	jnm.Mu.Lock()
	defer jnm.Mu.Unlock()
	delete(jnm.Map,key)
}
func (jnm *JobsNameMap) Members() map[string]string {
	jnm.Mu.Lock()
	defer jnm.Mu.Unlock()
	return jnm.Map
}


