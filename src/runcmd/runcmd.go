package runcmd
import(
	"myutils"
	"regexp"
	"strings"
	"io/ioutil"
	"time"
	"path"
	"fmt"
	"strconv"
	"os"
	"io"
	"net"
	"redisdb"
	"os/exec"
)
type StepRun struct {
	Sample 	  string
	Prefix 	  string
	Step      string 
	Index     string
	Command   string
	Status    string
	Service   string
	Db redisdb.Database
	SendTicker  *time.Ticker
	OutputStatus string
	DeployId string
	BaseDir string
	LogsQueue *myutils.StringQueue
	LogTicker *time.Ticker
	AliveTicker *time.Ticker
	SendRegistrySignal chan bool
}
func NewStepRun() *StepRun{
	sample := myutils.GetOsEnv("SAMPLE")
	redisAddr := myutils.GetOsEnv("ANGELINA_REDIS")
	db := redisdb.NewRedisDB("tcp",redisAddr)
	prefix := myutils.GetSamplePrefix(sample)
	deployId := myutils.GetOsEnv("DEPLOYMENTID")
	step := myutils.GetOsEnv("STEP")
	index := myutils.GetOsEnv("INDEX")
	service := myutils.GetOsEnv("SERVICE")
	baseDir := "/mnt/data"
	lgsq := myutils.NewStringQueue(1000)
	stick := time.NewTicker(30 * time.Second)
	logTicker := time.NewTicker(15 * time.Second)
	aliveTicker := time.NewTicker(60 * time.Second)
	return &StepRun{
		Sample: sample,
		Prefix: prefix,
		Step: step,
		Index: index,
		Command: "",
		Db: db,
		OutputStatus: "ready",
		Service: service,
		Status: "running",
		BaseDir: baseDir,
		DeployId: deployId,
		SendTicker: stick,
		LogsQueue: lgsq,
		SendRegistrySignal: make(chan bool),
		LogTicker: logTicker,
		AliveTicker: aliveTicker}
}
func (sr *StepRun) StartRun() {
	sr.DeleteDebugFile()
	if !sr.CheckStatus() {
		sr.SetCommand(sr.ReadCommand(sr.Step,sr.Index))
		go sr.ExecCmd(sr.Step,sr.Index)
	}else {
		sr.SetOutputStatus("succeed")
	}
	go sr.SendTickerFunc()
	go sr.SendAliveTickerFunc()
	sr.Db.RedisSubscribe("AngelinaRecoveryChan",sr.RegistryMe)
}
func (sr *StepRun) RegistryMe(data string) {
	if data == "WhoAlive" {
		sr.SendRegistrySignal <- true
	}
}
func (sr *StepRun) SendRegistryMsg() {
	sr.AppendLogToQueue("Info","receive registry message is WhoAlive")
	status := sr.SocketSendMessage(sr.CreateMsg("registry"))
	if status == false {
		sr.AppendLogToQueue("Error","send registry message failed.")
			
	}else {
		sr.AppendLogToQueue("Info","send registry message succeed.")
	}
} 
func (sr *StepRun) SetOutputStatus(status string) {
	sr.OutputStatus = status
	sr.Status = "finished"
}
func (sr *StepRun) CreateMsg(status string) string {
	data := `{"prefix":"%s","deployId":"%s","subStep":"%s","status":"%s"}`
	return fmt.Sprintf(data,sr.Prefix,sr.DeployId,sr.Step + "-" + sr.Index,status)
}
func (sr *StepRun) SendAliveTickerFunc() {
	for {
		select {
			case <- sr.AliveTicker.C:
				sr.SocketSendMessage(sr.CreateMsg("alive"))
			case <- sr.LogTicker.C:
				sr.WriteLogs()
		}
	}
}

func (sr *StepRun) SendTickerFunc() {
	for {
		select {
			case <- sr.SendTicker.C:
				sr.SendStatus()
			case <- sr.SendRegistrySignal:
				sr.SendRegistryMsg() 
		}
	}
}
func (sr *StepRun) SendStatus() {
	if sr.Status == "finished" && sr.OutputStatus != "ready" {
		sr.SocketSendMessage(sr.CreateMsg(sr.OutputStatus))
	}
}
func (sr *StepRun) SocketSendMessage(info string) bool {
	udpAddr,err := net.ResolveUDPAddr("udp4",sr.Service)
	if err != nil {
		sr.AppendLogToQueue("Error","resolve udp socket failed,reason: ",err.Error())
		return false
	}
	conn,err := net.DialUDP("udp",nil,udpAddr)
	if err != nil {
		sr.AppendLogToQueue("Error","dial udp failed,reason: ",err.Error())
		return false
	}
	_,err = conn.Write([]byte(info))
	if err != nil {
		sr.AppendLogToQueue("Error","write udp data failed,reason: ",err.Error())
		return false
	}
	var buf [512]byte
	n,err := conn.Read(buf[0:])
	if err != nil {
		sr.AppendLogToQueue("Error","read buffer failed from udp,reason: ",err.Error())
		return false
	}
	if string(buf[0:n]) != "received" {
		sr.AppendLogToQueue("Error","read received message is not received ")
		return false
	}
	return true
}
func (sr *StepRun) SetCommand(cmd string) {
	sr.Command = cmd
}
func (sr *StepRun) ReadCommand(step,index string) string {
	file := path.Join(sr.BaseDir,sr.Sample,step,".command")
	data,err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	info := strings.Split(string(data),"-***-")
	cmd,_ := strconv.Atoi(index)
	if cmd >= len(info) {
		return ""
	}
	return info[cmd]
}
func (sr *StepRun) CheckStatus() bool {
	regstr := sr.Step + "-" + sr.Index + ":.*" + "succeed"
	reg := regexp.MustCompile(regstr)
	file := path.Join(sr.BaseDir,sr.Sample,sr.Step,".status")
	_,err := os.Stat(file)
	if err != nil {
		return false
	}
	data,err := ioutil.ReadFile(file)
	info := string(data)
	if reg.FindString(info) != "" {
		return true
	}
	return false
}
func (sr *StepRun) ExecCmd(step,index string) {
	var status string
	if sr.Command == "" {
		status = "failed"
		sr.AppendLogToQueue("Error","execute command failed,because we don't read the command from .command file")
	}else {
		dir := path.Join(sr.BaseDir,sr.Sample,step)
		os.Chdir(dir)
		outFile := path.Join(dir,"logs",step + "-"  + index + "-output.log")
		errorFile := path.Join(dir,"logs",step + "-" + index + "-error.log")
		status = sr.RunCmd(sr.Command,outFile,errorFile)
		sr.AppendLogToQueue("Info","the command ",sr.Command," will run")
	}
	sr.SetOutputStatus(status)
	sr.AppendLogToQueue("Info","the command run status ",sr.Command," has send to channel")
}
func (sr *StepRun) RunCmd(cmdStr,outlog,errorlog string) (string) {
	_,errSt := os.Stat(errorlog)
    if errSt == nil {
        os.Remove(errorlog)
    }
	cmd := exec.Command("/bin/sh","-c",cmdStr)
	stdout, err := os.OpenFile(outlog, os.O_CREATE|os.O_WRONLY, 0600)
	defer stdout.Close()
	if err != nil {
		sr.AppendLogToQueue("Error","create output log failed,command execute failed,reason: ",err.Error())
		return "failed"
	}
	stderr, err := os.OpenFile(errorlog, os.O_CREATE|os.O_WRONLY, 0600)
    defer stderr.Close()
	if err != nil {
		sr.AppendLogToQueue("Error","create error log failed,command execute failed,reason: ",err.Error())
		return "failed"
	}
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	cmd.Start()
	err3 := cmd.Wait()
	if err3 != nil {
		sr.AppendLogToQueue("Error","command execute error,reason: ",err3.Error())
		return "failed"
	}
	return "succeed" 
}
func (sr *StepRun) AppendLogToQueue(level string,logStr ...string) {
	mystr := myutils.GetTime() 
	mystr = mystr + "\t" + level + "\t"
	for _,val := range logStr {
		mystr = mystr +  val 
	}
	mystr = mystr + "\n"
	sr.LogsQueue.PushToQueue(mystr)	
}
func (sr *StepRun) WriteLogs() {
	data := strings.Join(sr.LogsQueue.PopAllFromQueue(),"")
	logfile := path.Join(sr.BaseDir,sr.Sample,sr.Step,"logs", sr.Step + "-" + sr.Index + "-debug.log")
	var f *os.File
	var err error
	if checkFile(logfile) {
		f,err = os.OpenFile(logfile,os.O_APPEND | os.O_WRONLY,os.ModeAppend)
	}else {
		f,err = os.Create(logfile)
	}
	if err != nil {
		sr.AppendLogToQueue("Error",err.Error())
		return 
	}
	_,err1 := io.WriteString(f,data)
	if err1 != nil {
		sr.AppendLogToQueue("Error",err1.Error())
	}
}
func (sr *StepRun) DeleteDebugFile() {
	debugFile := path.Join(sr.BaseDir,sr.Sample,sr.Step,"logs", sr.Step + "-" + sr.Index + "-debug.log")
	_,err := os.Stat(debugFile)
	if err == nil {
		os.Remove(debugFile)
	}
}
func checkFile(filename string) bool {
	var exist = true
	if _,err := os.Stat(filename);os.IsNotExist(err) {

		exist = false
	}
	return exist
}
