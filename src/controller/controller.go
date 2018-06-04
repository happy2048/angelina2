package controller
import(
	"myutils"
	"encoding/json"
	"fmt"
	"time"
	"gopkg.in/gomail.v2"
	"strings"
)
type RunnerMessage struct {
    Prefix string `json:"prefix"`
    DeployId string `json:"deployId"`
    SubStep  string  `json:"subStep"`
    Status  string  `json:"status"`
}
type BackJobs struct {
	Waiting []string `json:"waiting"`
	Running []string `json:"running"`
	Starting []string `json:"starting"`
}
func (ctrl *Controller) Start() {
	go ctrl.ListenSocketService()
	go ctrl.HttpServer()
	go ctrl.RecoveryJobs()
	go ctrl.RecoverySendEmailJobs()
	time.Sleep(25 * time.Second)
	ctrl.CheckWhoAlive()
	ctrl.HandleRecoveryData()
	go ctrl.DeleteRecoveryKey()
	go ctrl.MyTickerFunc()
	go ctrl.TickerSendEmail()
	ctrl.StatChangeChan()
}
func (ctrl *Controller) PickJobStepToRun() {
	for _,jobName := range ctrl.RunningJobs.Members() {
		jobId := myutils.GetSamplePrefix(jobName)
		job := ctrl.JobsPool.Read(jobId)
		if job != nil {
			job.PickStepToRun()
		}else {
			ctrl.AppendLogToQueue("Error","job",jobName,"is nil")
		}
	}
}
func (ctrl *Controller) FlashJobStepStatus() {
	for _,jobName := range ctrl.RunningJobs.Members() {
		jobId := myutils.GetSamplePrefix(jobName)
		job := ctrl.JobsPool.Read(jobId) 
		if job != nil {
			go job.SendStepStatus(false)
		}
	}
}
func (ctrl *Controller) CheckJobStepAlive() {
	for _,jobName := range ctrl.RunningJobs.Members() {
		jobId := myutils.GetSamplePrefix(jobName)
		job := ctrl.JobsPool.Read(jobId) 
		if job != nil {
			go job.CheckDeployIsAlive()
		}
	}
}
func (ctrl *Controller) PushMessage(data string) {
	ctrl.AppendLogToQueue("Info","get runner message:",data)
	go ctrl.MessageQueue.PushToQueue(data)
}
func (ctrl *Controller) DeleteJob(job string) {
	if ctrl.WaitingRunJobs.Contains(job) && ctrl.WaitingRunJobs.Timestamp(job) < ctrl.WaitingDeleteJobs.Timestamp(job) {
		ctrl.WaitingRunJobs.Remove(job)
		ctrl.WaitingDeleteJobs.Remove(job)
		return
	}
	if ctrl.StartingJobs.Contains(job) {
		return 
	}
	if ctrl.RunningJobs.Contains(job) {
		prefix := myutils.GetSamplePrefix(job)
		ctrl.CancelJobs.Add(job)
		go ctrl.JobsPool.Read(prefix).DeleteCons()
		return
	}
	if ctrl.DeletingJobs.Contains(job) {
		ctrl.WaitingDeleteJobs.Remove(job)
		return
	}
	prefix := myutils.GetSamplePrefix(job)
	if ctrl.JobsPool.Contain(prefix) == false {
		ctrl.WaitingDeleteJobs.Remove(job)
		return
	}
	ctrl.WaitingDeleteJobs.Remove(job)

}
func  (ctrl *Controller) RoundHandleJob() {
	for _,job := range ctrl.WaitingDeleteJobs.Members() {
		go ctrl.DeleteJob(job)
	}
	for _,job := range ctrl.WaitingRunJobs.Members() {
		go ctrl.CreateJob(job)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ctrl *Controller) CreateJob(job string) {
	ctrl.AppendLogToQueue("Info","start to create job",job)
	prefix := myutils.GetSamplePrefix(job)
	ctrl.FinishedJobs.Remove(prefix)
	ctrl.SendMailJobs.Remove(job + "-*-" + "failed")
	ctrl.SendMailJobs.Remove(job + "-*-" + "succeed")
	if ctrl.RunningJobs.Contains(job) || ctrl.StartingJobs.Contains(job) {
		ctrl.AppendLogToQueue("Info","job",job,"has in the RunningJobs or StartingJobs")
		ctrl.WaitingRunJobs.Remove(job)
		return
	}
	if ctrl.DeletingJobs.Contains(job) {
		ctrl.AppendLogToQueue("Info","job",job,"is deleting,we create it after a while")
		return 
	}
	if ctrl.WaitingDeleteJobs.Contains(job) || ctrl.WaitingDeleteJobs.Timestamp(job) > ctrl.WaitingRunJobs.Timestamp(job) {
		ctrl.AppendLogToQueue("Info","job",job,"will be deleted")
		ctrl.WaitingRunJobs.Remove(job)
		return
	}
	ctrl.WaitingRunJobs.Remove(job)
	ctrl.AppendLogToQueue("Info","remove the job",job,"from WaitingRunJobs succeed")
	ctrl.StartingJobs.Add(job)
	ctrl.AppendLogToQueue("Info","add the job",job,"to StartingJobs succeed")
	myjob,err := NewJob(ctrl.RedisAddr,job,ctrl.FinishedSignal,ctrl.KubeConfig,ctrl.DeleteLocker,ctrl.RecoveryMap)
	if err != nil {
		ctrl.AppendLogToQueue("Error","job",job,"create failed,reason:",err.Error())
		tdata := &SimpleJob{
			Name: job,
			FinishedTime: time.Now(),
			Status: "failed",
			Log: "create job failed,reason:" + err.Error()}
		ctrl.StartingJobs.Remove(job)
		ctrl.AppendLogToQueue("Info","remove the job",job,"from StartingJobs succeed")
		ctrl.FinishedJobs.Add(myutils.GetSamplePrefix(job),tdata)
		ctrl.AppendLogToQueue("Info","add the job",job,"to FinishedJobs succeed")
		if ctrl.SmtpEnabled == true {
			ctrl.SendMailJobs.Add(job + "-*-" + "failed")
		}
		return
	}
	err1 := myjob.Start()
	if err1 != nil {
		ctrl.JobsPool.Write(myutils.GetSamplePrefix(job),myjob)
		ctrl.StartingJobs.Remove(job)
		ctrl.RunningJobs.Add(job)
		myjob.DeleteCons()
		return 
	}
	ctrl.AppendLogToQueue("Info","job",job,"is starting")
	ctrl.JobsPool.Write(myutils.GetSamplePrefix(job),myjob)
	ctrl.AppendLogToQueue("Info","job",job,"add to JobsPool succeed")
	ctrl.StartingJobs.Remove(job)
	ctrl.AppendLogToQueue("Info","job",job,"remove from StartingJobs succeed")
	ctrl.RunningJobs.Add(job)
	ctrl.AppendLogToQueue("Info","job",job,"add to RunningJobs succeed")
}
func (ctrl *Controller) HandleRecoveryData() {
	members := ctrl.MessageQueue.PopAllFromQueue()
	for _,info := range members {
		var data RunnerMessage
		err := json.Unmarshal([]byte(info),&data)
		if err != nil {
			ctrl.AppendLogToQueue("Error","json parse messge",info,"failed,reason:",err.Error())
			continue
		}
		if data.Prefix == "" || data.DeployId == "" || data.SubStep == "" || data.Status == "" {
			continue
		}
		if data.Status != "registry" {
			continue
		}
		key := data.Prefix + "-***-" + data.DeployId + "-***-" + data.SubStep + "-***-" + data.Status
		ctrl.RecoveryMap.Add(key)
	}
}
func (ctrl *Controller) RoundHandleRunnerData() {
	members := ctrl.MessageQueue.PopAllFromQueue()
	count := 0
	for _,info := range members {
		var data RunnerMessage
		err := json.Unmarshal([]byte(info),&data)
		if err != nil {
			ctrl.AppendLogToQueue("Error","json parse messge",info,"failed,reason:",err.Error())
			continue
		}
		if data.Prefix == "" || data.DeployId == "" || data.SubStep == "" || data.Status == "" {
			continue
		}
		if ctrl.JobsPool.Contain(data.Prefix) == false {
			delete := false
			name := ctrl.NameMap.Read(data.Prefix)
			if name == "" {
				delete  = true
			}else if ctrl.RunningJobs.Contains(name) == false && ctrl.DeletingJobs.Contains(name) == false{
					delete = true
			}
			if delete {
				ctrl.Kube.DeleteDeployment(data.DeployId)
			}
			continue
		}
		job := ctrl.JobsPool.Read(data.Prefix)
		tdata := strings.Split(data.SubStep,"-")
		if len(tdata) != 2 {
			continue
		}
		step := tdata[0]
		indexString := tdata[1]
		index := StringToInt(indexString)
		if !job.RunningDeployment.Contains(data.DeployId) {
			continue
		}
		if !job.Steps.Contains(step) {
			continue
		}
		if index >= len(job.Steps.Read(step).SubSteps) {
			continue
		}
		go job.HandleData(data.DeployId,data.SubStep,data.Status)
		if count % 10 == 0 {
			time.Sleep(3 * time.Second)
		}else {
			count++
		}
	}

}

func (ctrl *Controller) DeleteExpirationJob() {
	for key,job := range ctrl.FinishedJobs.Members() {
		dur := time.Since(job.FinishedTime)
		timeOut := time.Duration(86400) * time.Second
		if dur > timeOut {
			ctrl.FinishedJobs.Remove(key)
		}
	}
}

func (ctrl *Controller) GetAllJobs() string {
	redata := make([]string,0,1000) 
	split := "   "
	name := strings.Repeat(" ",32) + "Angelina" + strings.Repeat(" ",36)
	headLine := strings.Repeat("*",80)
	title := NormString("Date       Time",19) + split + NormString("Job Id",14) + split + NormString("Status",12) + split + "Job Name"
	thinLine := strings.Repeat("-",80)
	redata = append(redata,name)
	redata = append(redata,headLine)
	redata = append(redata,title)
	redata = append(redata,thinLine)
	for _,job := range ctrl.WaitingRunJobs.Members() {
		redata = append(redata,NormJobPrint(job,"WaitToRun"))
	}
	for _,job := range ctrl.WaitingDeleteJobs.Members() {
		redata = append(redata,NormJobPrint(job,"WaitToDelete"))
	}
	for _,job := range ctrl.StartingJobs.Members() {
		redata = append(redata,NormJobPrint(job,"Starting"))
	}
	for _,job := range ctrl.RunningJobs.Members() {
		redata = append(redata,NormJobPrint(job,"Running"))
	}
	for _,job := range ctrl.DeletingJobs.Members() {
		redata = append(redata,NormJobPrint(job,"Deleting"))
	}
	for _,job := range ctrl.FinishedJobs.Members() {
		if job != nil {
			redata = append(redata,NormJobPrint(job.Name,job.Status))
		}
	}
	redata = append(redata,headLine)
	return strings.Join(redata,"\n") + "\n"
}
func NormJobPrint(job,status string) string {
 	split := "   "
	line := myutils.GetTime() + split + myutils.GetSamplePrefix(job) + split + NormString(status,12) + split + job
	return line
}
func (ctrl *Controller) BackupJobs() {
	waiting := ctrl.WaitingRunJobs.Members()
	running := ctrl.RunningJobs.Members()
	starting := ctrl.StartingJobs.Members()
	redata := &BackJobs {
		Waiting: waiting,
		Starting: starting,
		Running: running}
	rest,err := json.Marshal(redata)
	if err != nil {
		ctrl.AppendLogToQueue("Error","json marshal backup data failed,reason:",err.Error())
		return 
	}
	ctrl.Db.RedisStringSetWithEx(ctrl.BackupKey,string(rest),86400)
}
func (ctrl *Controller) CheckWhoAlive() {
	for i := 0;i < 3;i++ {
		ctrl.Db.RedisPublish("AngelinaRecoveryChan","WhoAlive")
		time.Sleep(10 * time.Second)
	}
}
func (ctrl *Controller) DeleteRecoveryKey() {
	time.Sleep(60 * time.Second)
	for _,key := range ctrl.RecoveryMap.Members() {
		ctrl.RecoveryMap.Remove(key)
	}

}
func (ctrl *Controller) RecoveryJobs() {
	var back BackJobs
	data,err := ctrl.Db.RedisStringGet(ctrl.BackupKey)
	if err != nil {
		return 
	}
	err = json.Unmarshal([]byte(data),&back)
	if err != nil {
		return
	}
	for _, job := range back.Running {
		id := myutils.GetSamplePrefix(job)
        ctrl.NameMap.Add(id,job)
		ctrl.WaitingRunJobs.Add(job)
	}
	for _, job := range back.Starting {
		id := myutils.GetSamplePrefix(job)
        ctrl.NameMap.Add(id,job)
		ctrl.WaitingRunJobs.Add(job)
	}
	for _, job := range back.Waiting {
		id := myutils.GetSamplePrefix(job)
        ctrl.NameMap.Add(id,job)
		ctrl.WaitingRunJobs.Add(job)
	}
}
func (ctrl *Controller) GetJobStatus(key string) string {
	key = strings.Trim(key," ")
	if strings.Index(key,"pipe") == 0 {
		name := ctrl.NameMap.Read(key)
		return ctrl.GetJobStat(name,key)
	}else {
		return ctrl.GetJobStat(key,key)
	}

}

func (ctrl *Controller) GetJobStat(name,key string) string {
	split := "   "
	if name != "" {
		if ctrl.WaitingRunJobs.Contains(name) {
			return myutils.GetTime() + split + "Info" + split + "the job " + key + " is waiting to run"
		}
		if ctrl.RunningJobs.Contains(name) {
			prefix := myutils.GetSamplePrefix(name)
			job := ctrl.JobsPool.Read(prefix)
			if job != nil {
				return job.StepStatus + "\n" 
			}else {
				return myutils.GetTime() + split + "Info" + split + "get job " + key + " status failed"
			}
		}
		if ctrl.StartingJobs.Contains(name) {
			return myutils.GetTime() + split + "Info" + split + "job " + key + " is starting"
		}
		if ctrl.DeletingJobs.Contains(name) {
			return myutils.GetTime() + split + "Info" + split + "job " + key + " is deleting"
		}
		if ctrl.WaitingDeleteJobs.Contains(name) {
			return myutils.GetTime() + split + "Info" + split + "job " + key + " is waiting to delete"
		}
		prefix := myutils.GetSamplePrefix(name)
		if ctrl.FinishedJobs.Contain(prefix) {
			job := ctrl.FinishedJobs.Read(prefix)
			if job != nil {
				return job.Log + "\n"
			}else {
				return myutils.GetTime() + split + "Info" + split + "get job " + key + " status failed"
			}
		}
		return myutils.GetTime() + split + "Info" + split + "no this job id: " + key
	}else {
		return myutils.GetTime() + split + "Info" + split + "no this job id: " + key
	}
}
func (ctrl *Controller) CheckNameMap() {
	for key,name := range ctrl.NameMap.Members() {
		if ctrl.WaitingRunJobs.Contains(name) {
			continue
		}
		if ctrl.RunningJobs.Contains(name) {
			continue
		}
		if ctrl.StartingJobs.Contains(name) {
			continue
		}
		if ctrl.DeletingJobs.Contains(name) {
			continue
		}
		if ctrl.WaitingDeleteJobs.Contains(name) {
			continue
		}
		if ctrl.FinishedJobs.Contain(key) {
			continue
		}
		ctrl.NameMap.Remove(key)
	}
}
func (ctrl *Controller) AppendLogToQueue(level string,logStr ...string) {
	mystr := myutils.GetTime()
	mystr = mystr + "\t" + level + "\t"
	for _,val := range logStr {
        mystr = mystr + " " + val
    }
    ctrl.LogsQueue.PushToQueue(mystr)
}
func (ctrl *Controller) PrintInfo() {
	members := ctrl.LogsQueue.PopAllFromQueue()
	fmt.Println("***************************************start**********************************************")
	for _,data := range members {
		fmt.Println(data)
		time.Sleep(200 * time.Millisecond)
	}
	info := `
WaitingRunJobs:    %s
StartingJobs:      %s
RunningJobs:       %s
DeletingJobs:      %s
WaitingDeleteJobs: %s
FinishedJobs:      %s`
	wait := strings.Join(ctrl.WaitingRunJobs.Members(),",")
	start := strings.Join(ctrl.StartingJobs.Members(),",")
	run := strings.Join(ctrl.RunningJobs.Members(),",")
	deleting := strings.Join(ctrl.DeletingJobs.Members(),",")
	waitDelete := strings.Join(ctrl.WaitingDeleteJobs.Members(),",")
	finished := strings.Join(ctrl.FinishedJobs.Names(),",")
	fmt.Printf(info + "\n",wait,start,run,deleting,waitDelete,finished)
	fmt.Println("*****************************************end**********************************************")
}
func (ctrl *Controller) WriteLogs() {
	for _,name := range  ctrl.RunningJobs.Members() {
		prefix := myutils.GetSamplePrefix(name)
		 if ctrl.JobsPool.Contain(prefix) {
			job := ctrl.JobsPool.Read(prefix)
			if job != nil {
				go job.WriteLogs()
			}
		}
	}
}
func (ctrl *Controller) SendEmail(content string) (error) {
    m := gomail.NewMessage()
    m.SetHeader("From", ctrl.SendEmailInfo.User)
    m.SetHeader("To",ctrl.SendEmailInfo.To...)
    m.SetAddressHeader("Cc",ctrl.SendEmailInfo.User,ctrl.SendEmailInfo.User)
    m.SetHeader("Subject", "Angelina Job Runing Status")
    m.SetBody("text/html", content)
    d := gomail.NewDialer(ctrl.SendEmailInfo.SmtpServer,ctrl.SendEmailInfo.Port,ctrl.SendEmailInfo.User, ctrl.SendEmailInfo.Passwd)
    return d.DialAndSend(m)
}
func (ctrl *Controller) SendAllJobEmailInfo(members []string) {
	if len(members) == 0 {
		return 
	}
	tableTemp := `<table><tr><th>JobName</th><th>JobStatus</th></tr>%s</table>`
	lineTemp := `<tr><td>%s</td><td>%s</td></tr>`
	tmpstr := make([]string,0,len(members))
    nameMap := make(map[string]bool)
	for _,job := range members {
		infos := strings.Split(job,"-*-") 
		if len(infos) != 2 {
			nameMap[job] = true
			ctrl.SendMailJobs.Remove(job)
			continue
		}
		name := infos[0]
		status := infos[1]
		tmpstr = append(tmpstr,fmt.Sprintf(lineTemp,name,status))
		ctrl.SendMailJobs.Remove(job)
	}
	data := fmt.Sprintf(tableTemp,strings.Join(tmpstr,""))
	err := ctrl.SendEmail(data)
	if err != nil {
		ctrl.AppendLogToQueue("Error","send email failed,reason: ",err.Error())
		for _,job := range members {
			if _,ok := nameMap[job]; ok {
				continue
			}
			ctrl.SendMailJobs.Add(job)
		}
	}
}
func (ctrl *Controller) SendJobsEmail() {
	members := ctrl.SendMailJobs.Members()
	ctrl.SendAllJobEmailInfo(members)
}
func (ctrl *Controller) BackupWaitSendEmailJobs() {
	if ctrl.SmtpEnabled == false {
		return 
	}
	members := ctrl.SendMailJobs.Members() 
	if len(members) == 0 {
		return 
	}
	data := strings.Join(members,"-**-")
	_,err := ctrl.Db.RedisStringSetWithEx("AngelinaWaitSendEmail",data,180)
	if err != nil {
		ctrl.AppendLogToQueue("Error","backup waitSendEmailJob failed,reason: ",err.Error())
	}

}
func (ctrl *Controller) RecoverySendEmailJobs() {
	if ctrl.SmtpEnabled == false {
		return 
	}
	info,err := ctrl.Db.RedisStringGet("AngelinaWaitSendEmail")
	if err != nil {
		return 
	}
	data := strings.Split(info,"-**-")
	ctrl.SendAllJobEmailInfo(data)
}
func (ctrl *Controller) TickerSendEmail() {
	if ctrl.SmtpEnabled == false {
		return 
	}
	for {
		select {
			case <- ctrl.SendEmailTicker.C:
				ctrl.SendJobsEmail()
		}
	}
}
