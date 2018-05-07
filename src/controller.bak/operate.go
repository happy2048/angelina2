package controller
import (
	"os"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"kube"
	"syscall"
    "os/signal"
	"io"
	"myutils"
	"path"
	"io/ioutil"
)
/*
	Start(): sample实例的启动函数
*/
func (ct *Controller) Start() {
	go ct.DeleteSignal()
	ct.AppendLogToQueue("Info","listening delete signal")
	go ct.SendMyStatus()
    ct.DeleteErrorDeploy()
    ct.DeleteDebugFile()
	if !ct.ValidateStep() {
		ct.Failed <- true
	}
    ct.CheckPreStatus()
	ct.CheckIfFinished()
    ct.StartStep()
	go ct.Db.RedisSubscribe(ct.ListenContainerMessageChan,ct.PushMessageToQueue)
    go ct.Db.RedisSubscribe(ct.ListenClient,ct.DeleteMyself)
	ct.MyTicker()
}
func (ct *Controller) GetDeployId(step string) string {
	return "deploy" + myutils.GetSha256(ct.SampleName + step)[0:9]
}
func (ct *Controller) DeleteMyself(data string) {
	info := strings.Split(data,"###")
	if len(info) == 2 && info[0] == ct.Prefix && info[1] == "delete" && ct.Status != "delete" {
		ct.Status = "delete"
		ct.DeleteCons()
	} 

}
/*
	StartStep(): 处理不需要依赖其他step的step
*/
func (ct *Controller) StartStep() {
	ct.AppendLogToQueue("Info","start to run step")
	// item in members like "step1-0,step1-1"
	members := ct.WaitingRunningSteps.Members()
	for _,hkey := range members {
		step := strings.Split(hkey,"-")[0]
		presteps := ct.Steps.Read(step).Presteps
		if len(presteps) == 0 && ct.Steps.Read(step).Status == "ready" {
			for ind,_ := range ct.Steps.Read(step).SubSteps {
				status := ct.CreateDeployment(step + "-" + IntToString(ind))
				if status {
					ct.AppendLogToQueue("Info","create deployment for ",hkey)
					ct.SetStepRunning(hkey)
				}
			}
		}
	}
}
/*
	SetStepRunning(): 设置step状态为running,同时设置step的开始时间和keepalive开始时间
*/
func (ct *Controller) SetStepRunning(subStep string) {
	step := strings.Split(subStep,"-")[0]
	indexString := strings.Split(subStep,"-")[1]
	index := StringToInt(indexString)
	ct.Steps.Read(step).SubSteps[index].StepStartTime = time.Now()
	ct.Steps.Read(step).SubSteps[index].LastAliveTime = time.Now()
    ct.Steps.Read(step).SubSteps[index].Status = "running"
	if ct.Steps.Read(step).Status == "ready"  {
		ct.Steps.Read(step).Status = "running"
	}
}
func (ct *Controller) ValidateStep() bool {
	for key,value := range ct.Steps.Members() {
		if len(value.Presteps) != 0 {
			for _,pre := range value.Presteps {
				if !ct.Steps.Contains(pre)  {
					ct.AppendLogToQueue("Error","prestep ","pre "," of ",key," is invalid.")
					return false
				} 
			}
		}
	}
	return true
}
/*
	CheckPreStatus(): 检查哪些step是已经运行成功了，如果运行成功了就不运行了
*/
func (ct *Controller) CheckPreStatus() {
    dir := path.Join(ct.BaseDir,ct.SampleName)
	// step like "step1,step2..."
	for i := 1;i < ct.Steps.Len() + 1;i++ {
		step := "step" + strconv.Itoa(i)
		presteps := ct.Steps.Read(step).Presteps
		isSucceed := true
		for _,pre := range presteps {
			if ct.Steps.Read(pre).Status == "ready" {
				isSucceed = false
				break
			}  
		}
		sfile := path.Join(dir,step,".status") 
		if isSucceed == false {
			os.Remove(sfile)
			continue
		}
        if _,err := os.Stat(sfile);err != nil {
			continue
		}
		data,err1 := ioutil.ReadFile(sfile)
		if err1 != nil {
			continue
		}
		tdata := strings.Trim(string(data),"\n")
		status := strings.Split(tdata,"\n")
		succeed := 0
		for _,sub := range status {
			sub = strings.Trim(sub,"\n")
			stat := strings.Split(sub,":")[1]
			if stat != "succeed" {
				continue
			}
			mystep := strings.Split(sub,":")[0]
			tstep := strings.Split(mystep,"-")[0]
			if tstep != step  {
				continue
			}
			index := StringToInt(strings.Split(mystep,"-")[1])
			if ct.Steps.Contains(step) && index < len(ct.Steps.Read(step).SubSteps) {
				ct.Steps.Read(step).SubSteps[index].Status = "succeed"
				ct.WaitingRunningSteps.Remove(mystep)
				ct.Steps.Read(step).SubSteps[index].DeployId = ct.GetDeployId(mystep)
				succeed++
			}
	
		}
		if succeed == len(ct.Steps.Read(step).SubSteps) {
			ct.Steps.Read(step).Status = "succeed"
		}
	
	}
}
/*
	CheckIfFinished(): 检查任务是否完成，标准是WaitingRunningSteps()长度是否为0
*/
func (ct *Controller) CheckIfFinished() {
	if ct.WaitingRunningSteps.Len() == 0 {
		ct.DeleteCons()
	}
}
/*
	CheckBalanceStatus(): 检查是否有平衡状态，如果有，需要打破它
*/
/*
	DeleteErrorDeploy(): 删除错误状态的deployment，这是可能是上次运行该sample失败留下的，在本次sample运行之前，需要删掉它
*/
func (ct *Controller) DeleteErrorDeploy() {
	sleep := false
    for key,value := range ct.Steps.Members() {
		for ind,_ := range value.SubSteps {
			deployId := ct.GetDeployId(key + "-" + strconv.Itoa(ind))
			if ct.Kube.DeploymentExist(deployId) != 1 {
				_,err := ct.Kube.DeleteDeployment(deployId)
				if err != nil {
					ct.AppendLogToQueue("Error","delete deployment ",deployId," error,reason: ",err.Error())
				}else {
					sleep = true
				}
			}
		}
    }
	if sleep == true {
		time.Sleep(45 * time.Second)
	}

}
/*
	DeleteDeployment(step string): 删除deployment,这个删除不是真正意义上的删除，如果该容器在接下来的step上还会用到，把它扔到空闲deployment队列里，以便下次使用。
*/

func (ct *Controller) DeleteDeployment(subStep string) {
	step := strings.Split(subStep,"-")[0]
    index := StringToInt(strings.Split(subStep,"-")[1])
	deployId := ct.Steps.Read(step).SubSteps[index].DeployId
    conName := ct.Steps.Read(step).Container
    conId :=  GetConHashKey(conName)
	// 删除未确认cmd
	delData := deployId + ":" + subStep + ":" + "start"
	ct.ConfirmSet.Remove(delData)
	if ct.RunningDeployment.Contains(ct.Steps.Read(step).SubSteps[index].DeployId) == false {
		return 
	}
	// 如果整个pipeline中只会使用到一次这个容器，那么直接删除
	if ct.DeploymentStatus.Read(conId).Count == 1  || ct.DeploymentStatus.Read(conId).IdleQueue.Len() > 0{
		status,_ := ct.Kube.DeleteDeployment(deployId)
		if status == true {
			ct.RunningDeployment.Remove(deployId)
			ct.AppendLogToQueue("Info","deployment ",deployId," will not be used by next steps,we delete it")
		}
	}else {
		// 如果该deployment对应的container的空闲数为0，那么加入空闲队列，每种队列至多有一个空闲deployment。
			ct.DeploymentStatus.Read(conId).IdleQueue.PushToQueue(deployId)
			ct.AppendLogToQueue("Info","deployment ",deployId," will be used by next steps,remain it")
	}

}
/*
	CreateDeployment(step string): 创建deployment
*/
// step like "step1-1"
func (ct *Controller) CreateDeployment(subStep string) bool{
	// step  with p no refix 
	step := strings.Split(subStep,"-")[0]
	indexString := strings.Split(subStep,"-")[1]
	index := StringToInt(strings.Split(subStep,"-")[1])
	deployId := ct.GetDeployId(subStep)
	conName := ct.Steps.Read(step).Container
	conId :=  GetConHashKey(conName)
	// 如果整个pipeline中只会用到这种容器一次，或者没有空闲容器，那么直接创建
	if ct.DeploymentStatus.Read(conId).Count == 1  || ct.DeploymentStatus.Read(conId).IdleQueue.Len() == 0 {
		// 直接创建容器
		deployArgs := &kube.CreateDeployArgs{
			Sample: ct.SampleName,
			Container: ct.Steps.Read(step).Container,
			Index: indexString,
			DeployId: deployId,
			Step: step}
		status,_ := ct.Kube.CreateDeployment(deployArgs)
		if status == true {
			ct.Steps.Read(step).SubSteps[index].DeployId = deployId
			ct.RunningDeployment.Add(deployId)
			ct.AppendLogToQueue("Info","create new deployment ",deployId," for ",subStep)
			return true
		}else {
			return false
		}
	}else {
			// 否则从空闲队列里拿一个deployment运行命令
			deployId := ct.DeploymentStatus.Read(conId).IdleQueue.PopFromQueue()
			ct.Steps.Read(step).SubSteps[index].DeployId = deployId
			ct.RunningDeployment.Add(deployId)
			// 从订阅通1道通知运行任务
			data := deployId + ":" + subStep + ":" + "start"
			ct.Db.RedisPublish(ct.SendMessageToContainer,data)
			ct.ConfirmSet.Add(data)
			ct.AppendLogToQueue("Info","use remain deployment ",deployId," for ",subStep)
			return true

	}
}
/*
	DeleteCons(): sample运行完成或者接到删除信号需要做的收尾工作
*/
func (ct *Controller) DeleteCons() {
	// 执行容器删除
	if ct.Status == "finished" {
		return 
	}
	members := ct.RunningDeployment.Members()
  	if len(members) != 0 {
   		for _,val := range members {
     		status,_ := ct.Kube.DeleteDeployment(val)
        	if status {
           			ct.RunningDeployment.Remove(val)
            }
       	}
  	}
	ct.WriteStateFile()
	ct.SendStepStatus(true)
	ct.SaveRunTime()
	ct.WriteLogs()
	ct.Status = "finished"
	ct.Db.RedisSetSremMember(ct.RedisRunningSampleSet,ct.SampleName)
	ct.Kube.DeleteDeployment(ct.Prefix)
}
func (ct *Controller) DeleteSignal() {
	signal.Notify(ct.DelConSignal,syscall.SIGHUP,syscall.SIGINT,syscall.SIGTERM,syscall.SIGTSTP)
	for {
		select {
			case <- ct.DelConSignal:
				ct.DeleteCons()
				os.Exit(127)
		}
	
	}

}
func (ct *Controller) SendMyStatus() {
	for {
		select {
			case <- ct.AliveTicker.C:
				ct.Db.RedisPublish(ct.SendMessageToClient,ct.Prefix + "###" + "alive")
			case <- ct.Failed:
				ct.Db.RedisPublish(ct.SendMessageToClient,ct.Prefix + "###" + "dead")
				return 
		}
	}
}	
func (ct *Controller) MyTicker() {
	for {
		select {
			case <- ct.Ticker5.C:
				ct.RoundToHandleData()
				ct.SetStepStatus()
			case <- ct.Ticker10.C:
				ct.SendStepStatus(false)
				ct.PickStepToRun()
			case <- ct.Ticker15.C:
				ct.SendMessageAgain()
			case <- ct.Ticker30.C:
				ct.CheckDeployIsAlive()
			case <- ct.Ticker60.C:
				ct.StartStep()
		}
	}
}
/*
	SendStepStatus():定时发送sample中各step状态信息到redis，以便客户端进行展示
*/
/*
	WriteStateFile(): 每一个step运行完成以后，需要把状态立即写到相应的目录当中
*/

func (ct *Controller) WriteStateFile() {
	dir := path.Join(ct.BaseDir,ct.SampleName)
	for key,value := range ct.Steps.Members() {
		file := path.Join(dir,key,".status")
		tarr := make([]string,0,len(value.SubSteps))
		for ind,val := range value.SubSteps {
			tarr = append(tarr,key + "-" + IntToString(ind) + ":" + val.Status)
		}
		data := strings.Join(tarr,"\n")
		ioutil.WriteFile(file,[]byte(data),0644)			
	}

}
/*
	AppendLogToQueue(level string,logStr ...string): 将日志信息加入到日志队列
*/
func (ct *Controller) AppendLogToQueue(level string,logStr ...string) {
	mystr := myutils.GetTime()
	mystr = mystr + "\t" + level + "\t"
	for _,val := range logStr {
        mystr = mystr  + val
    }
    mystr = mystr + "\n"
    ct.LogsQueue.PushToQueue(mystr)

}

/*
	WriteLogs(): 将日志队列里的信息写入日志文件
*/
func (ct *Controller) WriteLogs() {
    data := strings.Join(ct.LogsQueue.PopAllFromQueue(),"")
    logfile := path.Join(ct.BaseDir,ct.SampleName,"step0",".debug.log")
    var f *os.File
    var err error
    if checkFile(logfile) {
        f,err = os.OpenFile(logfile,os.O_APPEND | os.O_WRONLY,os.ModeAppend)
    }else {
        f,err = os.Create(logfile)
    }
    if err != nil {
        ct.AppendLogToQueue("Error",err.Error())
        return
    }
    _,err1 := io.WriteString(f,data)
    if err1 != nil {
        ct.AppendLogToQueue("Error",err1.Error())
    }
}
/*
	checkFile(filename string): 检查文件是否存在
*/
func checkFile(filename string) bool {
    var exist = true
    if _,err := os.Stat(filename);os.IsNotExist(err) {

        exist = false
    }
    return exist
}

/*
	DeleteDebugFile():每次运行sample之前需要删除之前的debug文件
*/
func (ct *Controller) DeleteDebugFile() {
    debugFile := path.Join(ct.BaseDir,ct.SampleName,"step0",".debug.log")
    _,err := os.Stat(debugFile)
    if err == nil {
        os.Remove(debugFile)
    }
}
/*
	CheckDeployIsAlive(): 检查deployment是否是alive
*/
func (ct *Controller) CheckDeployIsAlive() {
	for key,value := range ct.Steps.Members() {
		for ind,ival := range value.SubSteps {
			if ival.Status == "running" && ival.CheckAlive == false {
				go ct.RcreateDeployment(key,ind)
			}
		} 
	}
}
/*
	RcreateDeployment(step string): 如果超时，那么重新创建
*/
func (ct *Controller) RcreateDeployment(step string,index int) {
	ct.Steps.Read(step).SubSteps[index].CheckAlive = true
	defer func() {ct.Steps.Read(step).SubSteps[index].CheckAlive = false}()
	dur := time.Since(ct.Steps.Read(step).SubSteps[index].LastAliveTime)
	timeout := time.Duration(360) * time.Second
	if dur > timeout {
		ct.Steps.Read(step).SubSteps[index].LastAliveTime = time.Now()
		tid := ct.Steps.Read(step).SubSteps[index].DeployId  
        ct.Kube.DeleteDeployment(tid)
		subStep := step + "-" + IntToString(index)
		deployId := ct.GetDeployId(subStep)
		deployArgs := &kube.CreateDeployArgs{
        Sample: ct.SampleName,
		DeployId: deployId,
		Index: IntToString(index),
        Container: ct.Steps.Read(step).Container,
        Step: step}
		ct.RunningDeployment.Remove(tid)
		time.Sleep(60 * time.Second)
		if ct.Kube.DeploymentExist(tid) == kube.NotFound {
       		status,_ := ct.Kube.CreateDeployment(deployArgs)
			if status == true {
		  		ct.Steps.Read(step).SubSteps[index].DeployId = deployId
                ct.RunningDeployment.Add(deployId)
            	ct.AppendLogToQueue("Info","deployment ",tid," status is invalid,we replace it by ",deployId)
			}
		}
	}

}
/*
	SendMessageAgain(): 对于从通道发送执行命令给deployment需要检测对方是否接收到，如果没有确认消息返回，需要重新发送
*/

func (ct *Controller) SendMessageAgain() {
	if ct.ConfirmSet.Len() != 0 {
        for _,val := range ct.ConfirmSet.Members() {
            ct.AppendLogToQueue("Info","await acknowledge message: ",val)
            ct.Db.RedisPublish(ct.SendMessageToContainer,val)
        }
    }

}

func (ct *Controller) SaveRunTime() {
	succeed := 0
	all := 0
	for _,val := range ct.Steps.Members() {
		for _,ival := range val.SubSteps {
			all++
			if ival.Status == "succeed" && ival.StepRunTime != "0h 0m 0s" {
				succeed++
			}
		
		}
	}
	if succeed == all {
		if !myutils.CheckFileExist(path.Join(ct.BaseDir,ct.SampleName,"step0",".template"))  {
			return 
		}
	  	data,err := ioutil.ReadFile(path.Join(ct.BaseDir,ct.SampleName,"step0",".template"))	
	  	if err != nil {
			return
		}
		tdata := strings.Trim(string(data),"\n")
		pipeid := "pipeid" + myutils.GetSha256(strings.Trim(tdata," "))[:15]
		tm,err := ct.Db.RedisHashGet(pipeid,"estimate-time")
		if err != nil {
			ct.AppendLogToQueue("Error","save estimate time first query error: " + err.Error())
			tm,err = ct.Db.RedisHashGet(tdata,"estimate-time")
			if err != nil {
				ct.AppendLogToQueue("Error","save estimate time second query error: " + err.Error())
				return
			}
			pipeid = tdata
		}
		now := time.Now()
		nowUnix := now.Unix()
		startUnix := ct.StartTime.Unix()
		last,err := strconv.ParseInt(tm,10,64)	
		if err != nil {
			return			
		}
		if last == 0 {
			_,err1 := ct.Db.RedisHashSet(pipeid,"estimate-time",strconv.FormatInt(nowUnix - startUnix,10))
			if err1 != nil {
				return 
			}
			ct.AppendLogToQueue("Info","update time sucessed,new estimate time is: ",strconv.FormatInt(nowUnix - startUnix,10))
		}else {
			newTime := int64(float64(last) * 0.9 + 0.1 * float64(nowUnix - startUnix))
			_,err := ct.Db.RedisHashSet(pipeid,"estimate-time",strconv.FormatInt(newTime,10))
			if err != nil {
				return 
			}
			ct.AppendLogToQueue("Info","update time sucessed,new estimate time is: ",strconv.FormatInt(newTime,10))
		}
	}

}
/*
    HandleData(sample,deployId,subStep,status string): 处理消息的函数
*/
func (ct *Controller) HandleData(deployId,subStep,status string) {
	ct.AppendLogToQueue("Info","get message: ", deployId," ",subStep," ",status)
	step := strings.Split(subStep,"-")[0]
	indexString := strings.Split(subStep,"-")[1]
	index := StringToInt(indexString)
	if status == "received" {
		ct.ConfirmSet.Remove(deployId + ":" + subStep + ":" + "start")
		return 
	}
	if status == "alive" {
		if ct.Steps.Contains(step) && ct.Steps.Read(step).SubSteps[index].Status == "running" && ct.Steps.Read(step).SubSteps[index].DeployId == deployId {
			ct.Steps.Read(step).SubSteps[index].LastAliveTime = time.Now()
			ct.AppendLogToQueue("Info","deployment ",deployId," of sample ",ct.SampleName," is keeping alive.")
			return 
		}
	}
	if status != "succeed" && status != "failed" {
		return
	}
	reply := deployId + ":" + subStep + ":" + "received"
	ct.AppendLogToQueue("Info","send reply message: ",reply)
	ct.Db.RedisPublish(ct.SendMessageToContainer,reply)	
	if ct.WaitingRunningSteps.Contains(subStep) == false {
		return 
	}
	for _,pre := range ct.Steps.Read(step).Presteps {
		if ct.Steps.Read(pre).Status != "faild" && ct.Steps.Read(pre).Status != "succeed" {
			ct.AppendLogToQueue("Info","get invalid step status of ",subStep)
			return 
		}
	}
	ct.Steps.Read(step).SubSteps[index].Status = status 
	ct.AppendLogToQueue("Info","set status for ",subStep)
	statusFile := path.Join(ct.BaseDir,ct.SampleName,step,".status")
	ct.WriteStatusFile(statusFile,subStep + ":" + status)
	runTime := myutils.GetRunTime(ct.Steps.Read(step).SubSteps[index].StepStartTime)
	ct.Steps.Read(step).SubSteps[index].StepRunTime = runTime
	ct.DeleteDeployment(subStep)
	ct.WaitingRunningSteps.Remove(subStep)
	count := ct.WaitingRunningSteps.Len()
	if count == 0 {
		ct.DeleteCons()
		return
	}
}
func (ct *Controller) WriteStatusFile(file,data string) {
	ct.Mu.Lock()
	defer ct.Mu.Unlock()
	subStep := strings.Split(data,":")[0]
	info,err := os.Stat(file)
	if os.IsNotExist(err) {
		ioutil.WriteFile(file,[]byte(data + "\n"),0644)
	}else if err == nil {
		if info.Size() == 0 {
			ioutil.WriteFile(file,[]byte(data + "\n"),0644)
		}else {
			tdata,_ := ioutil.ReadFile(file)
			mydata := string(tdata)
			if strings.Index(mydata,subStep) != -1 {
				mydata = strings.Replace(mydata,subStep + ":" + "failed",data,-1)
				mydata = strings.Replace(mydata,subStep + ":" + "succeed",data,-1)
				ioutil.WriteFile(file,[]byte(mydata),0644)
			}else {
				mydata = mydata + data + "\n"
				ioutil.WriteFile(file,[]byte(mydata),0644)
			}
		}
	
	}

}
func (ct *Controller) SetStepStatus() {
	for _,value := range ct.Steps.Members() {
		succeed := 0
		failed := 0
		slen := len(value.SubSteps)
		for _,sub := range value.SubSteps {
			if sub.Status == "succeed" {
				succeed++
			}else if sub.Status == "failed" {
				failed++
			}
		}
		if slen == succeed {
			value.Status = "succeed"
		} else if succeed + failed == slen && failed > 0 {
			value.Status = "failed"
		}
	}

}
func (ct *Controller) PickStepToRun() {
	ct.SetStepStatus()
	members := ct.WaitingRunningSteps.Members()
	var sortMembers StringSort
	sortMembers = members
	sort.Sort(sortMembers)
	for _,subStep := range sortMembers {
		step := strings.Split(subStep,"-")[0]
		indexString := strings.Split(subStep,"-")[1]
		index := StringToInt(indexString)
		if ct.Steps.Read(step).SubSteps[index].Status == "running" {
			continue
		}
		presteps := ct.Steps.Read(step).Presteps
		if len(presteps) != 0 {
			succeed := 0
			failed := 0
			for _,pre := range presteps {
				state := ct.Steps.Read(pre).Status
				if state == "succeed" {
					succeed++
				}else if state == "failed" {
					failed++
				}
			}
			if succeed == len(presteps) {
				status := ct.CreateDeployment(subStep)
				if status {
					ct.SetStepRunning(subStep)
				}
			}
			if failed > 0 {
				ct.Steps.Read(step).SubSteps[index].Status = "failed"
				ct.WaitingRunningSteps.Remove(subStep)
			}
		}
	}
	count := ct.WaitingRunningSteps.Len()
	if count == 0 {
		ct.DeleteCons()
	}
}
func (ct *Controller) PushMessageToQueue(data string) {
	go ct.MessageQueue.PushToQueue(data)
}

func (ct *Controller) RoundToHandleData() {
		if ct.MessageQueue.Len() != 0 {
			members := ct.MessageQueue.PopAllFromQueue()
			for _,data := range members {
				info := strings.Split(data,":")
				if len(info) == 3 {
					deployId := info[0]
					subStep := info[1]
					step := strings.Split(subStep,"-")[0]
					indexString := strings.Split(subStep,"-")[1]
					index := StringToInt(indexString)
					status := info[2]
					if !ct.RunningDeployment.Contains(deployId) {
						return 
					}
					if !ct.Steps.Contains(step) {
						return
					}
					if index >= len(ct.Steps.Read(step).SubSteps) {
						return 
					}
					ct.HandleData(deployId,subStep,status)
				}
			}
		}
}
func (ct *Controller) SendStepStatus(writeFile bool) {
	status := ct.GetStepStatusString()
    ct.Db.RedisStringSetWithEx(ct.Prefix,status,180)
	if writeFile {
		fileName := path.Join(ct.BaseDir,ct.SampleName,"running-status.log")
		ioutil.WriteFile(fileName,[]byte(status),0644)
	}
}
func (ct *Controller) GetStepStatusString() string{
	header := `                                                      Running  Status                                             
*******************************************************************************************************************************
Software          Name: angelina
Software       Version: v2.0
Template          Name: TEMPLATE
Template Estimate Time: ESTIMATE
Running Sample    Name: %v
Already Running   Time: HAVARUNTIME
------------------------------------------------------------------------------------------------------------------------------
%v
------------------------------------------------------------------------------------------------------------------------------`
	var endstr = "*******************************************************************************************************************************"
	count := 0
	for _,val := range ct.Steps.Members() {
		count = count + len(val.SubSteps)
	}
	title := NormString("Date       Time",19) + "   " + NormLine("Step","Sub-Steps","Status","Deployment-Id","Run-Time","Pre-Steps","Command")
	header = fmt.Sprintf(header,ct.SampleName,title)
	store := make([]string,0,count)
	store = append(store,header)
	for i := 1;i < ct.Steps.Len() + 1;i++ {
		key := "step" + strconv.Itoa(i)
		value := ct.Steps.Read(key).SubSteps
		for ind,val := range value {
			subStep := strconv.Itoa(ind)
			status := val.Status
			deployId := val.DeployId
			cmd := ct.Steps.Read(key).Command
			var runTime string
			if status == "succeed" || status == "failed" {
				runTime = val.StepRunTime
			}else if status == "running" {
				runTime = myutils.GetRunTime(val.StepStartTime)
			}else {
				runTime = "0h 0m 0s"
			}
			preSteps := strings.Join(ct.Steps.Read(key).Presteps,",")
			preSteps = strings.Replace(preSteps,"step","",-1)
			preSteps = strings.Trim(preSteps,",")
			if preSteps == "" {
				preSteps = "---"
			}
			data := GetLine(key,subStep,status,deployId,runTime,preSteps,cmd)
			store = append(store,data)
		}
	}
	store = append(store,endstr)
	return strings.Join(store,"\n")
}
func NormString(info string,length int) string {
	llen := len(info)
	if llen <= length {
		return info + strings.Repeat(" ",length - llen) 
	}else {
		return info[0:length]
	}

}
func GetLine(step,subStep,status,deployId,runTime,preSteps,cmd string) string {
	return myutils.GetTime() + "   " + NormLine(step,subStep,status,deployId,runTime,preSteps,cmd)

}
func NormLine(step,subStep,status,deployId,runTime,preSteps,cmd string) string {
	split := "  "
	step = NormString(step,7)
	subStep = NormString(subStep,3)
	status = NormString(status,7)
	deployId = NormString(deployId,15)
	runTime = NormString(runTime,11)
	preSteps = NormString(preSteps,20)
	cmd = NormString(strings.Trim(cmd," "),25)
	return step + split + subStep + split  + status + split + deployId + split + runTime + split + preSteps + split + cmd
}
