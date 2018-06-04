package controller
import (
    "io"
	"io/ioutil"
    "net/http"
	"fmt"
	"strings"
	"strconv"
	"myutils"
	"encoding/json"
	gjson "github.com/tidwall/gjson"
)
/*
func main() {
	http.HandleFunc("/hello",JobsOptionServer)
	http.HandleFunc("/query",JobsOptionServer)
	err := http.ListenAndServe(":12345",nil)
	if err != nil {
		fmt.Println(err)
	}
}
*/
type Redata  struct {
	Code  int `json: "code"`
	Msg   string `json: "msg"`
	Data  string  `json: "data"`
}
func (ctrl *Controller) HttpServer() {
	http.HandleFunc("/job",ctrl.JobsOptionServe)
	http.HandleFunc("/checkJob",ctrl.QueryJobStatus)
	http.HandleFunc("/cancelJobs",ctrl.DeleteAllJobs)
	http.HandleFunc("/query",ctrl.QueryOptionServe)
	http.HandleFunc("/queryTemp",ctrl.ListAllTemp)
	http.HandleFunc("/deleteTemp",ctrl.TemplateDelete)
	http.HandleFunc("/getTemp",ctrl.TemplateGetContent)
	http.HandleFunc("/storeTemp",ctrl.StorePipeServe)
	http.HandleFunc("/checkTemp",ctrl.TemplateCheckIsExist)
	http.HandleFunc("/cancelEmails",ctrl.CancelSendCurrentEmails)
	http.HandleFunc("/angelina-runner",ctrl.DloadRunnerCmdServe)
	err := http.ListenAndServe(ctrl.Service,nil)
	if err != nil {
		myutils.Print("Error","create http server failed,reason: " + err.Error(),true)
	}

}
func (ctrl *Controller) DloadRunnerCmdServe(w http.ResponseWriter,req *http.Request) {
	if req.Method == "GET" {
		bdata,err := ioutil.ReadFile(ctrl.RunnerCmdPath)
		if err != nil {
			io.WriteString(w,err.Error())
			return
		}
		io.WriteString(w,string(bdata))
	}

}
func (ctrl *Controller) JobsOptionServe(w http.ResponseWriter,req *http.Request) {
	ctrl.HandleRequests(w,req)
}
func (ctrl *Controller) QueryOptionServe(w http.ResponseWriter,req *http.Request) {
	ctrl.HandleQuery(w,req)
}
func (ctrl *Controller) HandleQuery(w http.ResponseWriter,req *http.Request) {
	if req.Method == "GET" { 
		data := ctrl.GetAllJobs()
		ctrl.ReturnValue(w,1000,data,"get data ok")
	}else {
		ctrl.ReturnValue(w,1000,"","invalid url method ")
	}
}
func (ctrl *Controller) HandleRequests(w http.ResponseWriter,req *http.Request) {
	if req.Method == "GET" || req.Method == "POST" {
		req.ParseForm()
		if len(req.Form["job"]) != 1 {
			ctrl.ReturnValue(w,1101,"","invalid url,no job given or job gives more than one.")
			return 
		}
		if len(req.Form["operate"]) != 1 {
			ctrl.ReturnValue(w,1102,"","invalid url,no option given or option gives more than one.")
			return 
		}
		if req.Method == "POST" && req.Form["operate"][0] == "delete"  {
			name := strings.Trim(req.Form["job"][0]," ")
			if strings.Index(name,"pipe") == 0 {
				job := ctrl.NameMap.Read(name)
				if job == "" {
					ctrl.ReturnValue(w,1000,"","invalid job id " + name )
					return	
				} 
				name = job
			}
			ctrl.WaitingDeleteJobs.Add(name)
			ctrl.ReturnValue(w,1000,"",name + " will be deleted.")
			return 
		}  
		if req.Method == "GET" && req.Form["operate"][0] == "create" {
			name := strings.Trim(req.Form["job"][0]," ")
			ctrl.ReturnValue(w,1000,"",name + " has received.")
			id := myutils.GetSamplePrefix(name)
			ctrl.NameMap.Add(id,name)
			ctrl.AppendLogToQueue("Info","get message to create job ",name)
			ctrl.WaitingRunJobs.Add(name)
			return
		}
		if req.Method == "GET" && req.Form["operate"][0] == "status" {
			data := ctrl.GetJobStatus(req.Form["job"][0])
			ctrl.ReturnValue(w,1000,data,"get status ok")
			return 
		}
		ctrl.ReturnValue(w,1103,"","invalid url request.")
		 
	}else {
		msg := fmt.Sprintf(`invalid method %s for path %s`,req.Method,req.URL.Path)
		ctrl.ReturnValue(w,1100,"",msg)
	}
}

func (ctrl *Controller) ReturnValue(w http.ResponseWriter,code int,data,msg string) {
	redata,err := json.Marshal(&Redata{Code: code,Msg: msg,Data: data})
	if err != nil {
		ctrl.AppendLogToQueue("Error","marshal json return message failed,reason: ",err.Error())
		return 
	}
	io.WriteString(w,string(redata))
}
func (ctrl *Controller) ListAllTemp(w http.ResponseWriter,req *http.Request) {
	if req.Method != "GET" {
		ctrl.ReturnValue(w,1201,"","invalid method " + req.Method)
		return
	}
	redata := make([]string,0,1000)
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	members,err := ctrl.Db.RedisSetMembers(redisKey)
	if err != nil {
		ctrl.ReturnValue(w,1202,"","read template pool failed,reason" + err.Error())
		return 
	}
	header := fmt.Sprintf("%s\t%s\t%s\t%s",NormString("Pipeline Id",21),NormString("Pipeline Name",25),NormString("Estimate Time",20),"Pipeline Description")
	redata = append(redata,header)
	for _,pid := range members {
		name,err := ctrl.Db.RedisHashGet(pid,"pipeline-name")
		if err != nil {
			ctrl.AppendLogToQueue("Error","get pipeline",pid,"failed,reason: ",err.Error())
			continue
		}
		desc,err := ctrl.Db.RedisHashGet(pid,"pipeline-description")
		if err != nil {
			ctrl.AppendLogToQueue("Error","get pipeline",pid,"failed,reason: ",err.Error())
			continue
		}
		tm,err := ctrl.Db.RedisHashGet(pid,"estimate-time")
		if err != nil {
			ctrl.AppendLogToQueue("Error","get pipeline",pid,"failed,reason: ",err.Error())
			continue
		}
		tint,err := strconv.ParseInt(tm,10,64)
		if err != nil {
			ctrl.AppendLogToQueue("Error","get pipeline",pid,"failed,reason: ",err.Error())
			continue
		}
		tmstr := myutils.GetRunTimeWithSeconds(tint)
		line := fmt.Sprintf("%s\t%s\t%s\t%s",NormString(pid,21),NormString(name,25),NormString(tmstr,20),desc)
		redata = append(redata,line)
		
	}
	ctrl.ReturnValue(w,1000,strings.Join(redata,"\n"),"query ok")
}

func (ctrl *Controller) DisplayPipeline(w http.ResponseWriter,info string) {
	data,err := ctrl.GetPipelineContent(info)
	if err == nil {
        ctrl.ReturnValue(w,1000,data,"query ok")
		return 
	}
	data,err = ctrl.Db.RedisHashGet(info,"pipeline-content")
	if err != nil {
        ctrl.ReturnValue(w,1000,"","Error: query template " + info + " failed,reason: " + err.Error())
		return 
	}
    ctrl.ReturnValue(w,1000,data,"query ok")
}
func (ctrl *Controller) GetPipelineContent(name string) (string,error) {
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	data,err := ctrl.Db.RedisHashGet(pipeid,"pipeline-content")
	if  err == nil {
		return data,nil
	}
	data,err = ctrl.Db.RedisHashGet(strings.Trim(name," "),"pipeline-content")
	return data,err
}
func (ctrl *Controller) CheckPipelineExist(w http.ResponseWriter,name string) {
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	status,err := ctrl.Db.RedisSetSisMember(redisKey,pipeid)
	if err == nil {
		if status == true {
			ctrl.ReturnValue(w,1000,"true","")
		
		}else {
			ctrl.ReturnValue(w,1000,"false","")
		}
		return
	}
	status,err = ctrl.Db.RedisSetSisMember(redisKey,strings.Trim(name," "))
	if err != nil {
		ctrl.AppendLogToQueue("Error","check",name,"failed,reason: ",err.Error())
		ctrl.ReturnValue(w,1207,"false","check " + name + " failed,reason: " + err.Error())
		return
	}
	if status == true {
		ctrl.ReturnValue(w,1000,"true","")
	}else {
		ctrl.ReturnValue(w,1000,"false","")
	}
}
func (ctrl *Controller) DeletePipeline(w http.ResponseWriter,name string) {
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	_,err := ctrl.Db.RedisSetSisMember(redisKey,pipeid)
	if err  == nil {
		ctrl.Db.RedisSetSremMember(redisKey,pipeid)
		ctrl.Db.RedisDelKey(pipeid)
		ctrl.ReturnValue(w,1000,"","delete ok")
		return 
	}
	ctrl.Db.RedisSetSremMember(redisKey,name)
	ctrl.Db.RedisDelKey(name)
	ctrl.ReturnValue(w,1000,"","delete ok")
}
func (ctrl *Controller) CheckJobIsExist(w http.ResponseWriter,name string) {
	status := "NotFound"
	if ctrl.WaitingRunJobs.Contains(name) {
		status = "WaitToRun"
	}else if ctrl.RunningJobs.Contains(name) {
		status = "Running"
	}else if ctrl.StartingJobs.Contains(name) {
        status = "Starting"
	}else if ctrl.DeletingJobs.Contains(name) {
        status = "Deleting"
	}else if ctrl.WaitingDeleteJobs.Contains(name) {
     	status = "WaitToDelete"   
	}
	ctrl.ReturnValue(w,1000,status,"query ok")
}
func (ctrl *Controller) TemplateDelete(w http.ResponseWriter,req *http.Request) {
	if req.Method != "DELETE"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	req.ParseForm()
	if len(req.Form["name"]) != 1 {
		ctrl.ReturnValue(w,1301,"","invalid url,no name given or name gives more than one.")
		return 
	}
	ctrl.DeletePipeline(w,req.Form["name"][0])
}
func (ctrl *Controller) TemplateGetContent(w http.ResponseWriter,req *http.Request) {
	if req.Method != "GET"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	req.ParseForm()
	if len(req.Form["name"]) != 1 {
		ctrl.ReturnValue(w,1301,"","invalid url,no name given or name gives more than one.")
		return 
	}
	ctrl.DisplayPipeline(w,req.Form["name"][0])

}
func (ctrl *Controller) TemplateCheckIsExist(w http.ResponseWriter,req *http.Request) {
	if req.Method != "GET"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	req.ParseForm()
	if len(req.Form["name"]) != 1 {
		ctrl.ReturnValue(w,1301,"","invalid url,no name given or name gives more than one.")
		return 
	}
	ctrl.CheckPipelineExist(w,req.Form["name"][0])
}
func (ctrl *Controller) QueryJobStatus(w http.ResponseWriter,req *http.Request) {
	if req.Method != "GET"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	req.ParseForm()
	if len(req.Form["name"]) != 1 {
		ctrl.ReturnValue(w,1301,"","invalid url,no name given or name gives more than one.")
		return 
	}
	ctrl.CheckJobIsExist(w,req.Form["name"][0])
}
func (ctrl *Controller) StorePipeServe(w http.ResponseWriter,req *http.Request) {
	if req.Method != "POST"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	body,err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctrl.ReturnValue(w,1301,"","ready pipeline template failed,please try again.")
		return 
	}
	bodyStr := string(body)
	ctrl.StorePipe(w,bodyStr)
}
func (ctrl *Controller) StorePipe(w http.ResponseWriter,info string) {
	pInfo := ctrl.ParsePipeline(info)
	if _,ok := pInfo["pipeline-name"]; !ok {
		ctrl.ReturnValue(w,1402,"","invalid pipeline template,no field \"pipeline-name\".")
		return 
	}
	if _,ok := pInfo["pipeline-description"]; !ok {
		ctrl.ReturnValue(w,1403,"","invalid pipeline template,no field \"pipeline-description\".")
		return 
	}
	if _,ok := pInfo["pipeline-content"]; !ok {
		ctrl.ReturnValue(w,1404,"","invalid pipeline template,no field \"pipeline-content\".")
		return 

	}
	name := pInfo["pipeline-name"]
	pdesc := pInfo["pipeline-description"]
	pcon := pInfo["pipeline-content"]
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	_,err := ctrl.Db.RedisSetAdd(redisKey,pipeid)
	if err != nil {
		ctrl.ReturnValue(w,1405,"","store pipeline template failed,reason: " + err.Error())
		return 
	}
	_,err = ctrl.Db.RedisHashSet(pipeid,"pipeline-name",name)
	if err != nil {
		ctrl.ReturnValue(w,1406,"","store pipeline template failed,reason: " + err.Error())
		return 
	} 
	_,err = ctrl.Db.RedisHashSet(pipeid,"pipeline-description",pdesc)
	if err != nil {
		ctrl.ReturnValue(w,1407,"","store pipeline template failed,reason: " + err.Error())
		return 
	}
	_,err = ctrl.Db.RedisHashSet(pipeid,"pipeline-content",pcon)
	if err != nil {
		ctrl.ReturnValue(w,1407,"","store pipeline template failed,reason: " + err.Error())
		return 
	}
	tm,_ := ctrl.Db.RedisHashGet(pipeid,"estimate-time")
	var storeTime string
	if tm != "" && tm != "0" {
		storeTime = tm
	}else {
		storeTime = "0"
	}
	_,err = ctrl.Db.RedisHashSet(pipeid,"estimate-time",storeTime)
	if err != nil {
		ctrl.ReturnValue(w,1408,"","store pipeline template failed,reason: " + err.Error())
		return 
	}
	ctrl.ReturnValue(w,1000,"","store pipeline template succeed.")
}
func (ctrl *Controller) ParsePipeline(data string) map[string]string {
	var pname string
	var pdesc string
	var pcon  string
	redata := make(map[string]string)
	if ! gjson.Valid(data) {
		return redata
	}
	jsonObj := gjson.Parse(data)
	jsonObj.ForEach(func(key,value gjson.Result)bool{
		if key.String() == "pipeline-name" {
			pname = strings.Trim(value.String()," ")
			if pname != "" {
				redata["pipeline-name"] = pname
			} 
		}else if key.String() == "pipeline-description" {
			pdesc = value.String()
			if pdesc != "" {
				redata["pipeline-description"] = pdesc
			}
		}else if key.String() == "pipeline-content" {
			pcon = strings.Trim(value.String()," ")
			if pcon != "" {
				redata["pipeline-content"] = pcon
			}
		}
		return true
	})
	return redata
}
func (ctrl *Controller) DeleteAllJobs(w http.ResponseWriter,req *http.Request) {
	if req.Method != "DELETE"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	for _,item := range ctrl.WaitingRunJobs.Members() {
		ctrl.WaitingDeleteJobs.Add(item)
	}
	for _,item := range ctrl.StartingJobs.Members() {
		ctrl.WaitingDeleteJobs.Add(item)
	}
	for _,item := range ctrl.RunningJobs.Members() {
		ctrl.WaitingDeleteJobs.Add(item)
	}
	ctrl.ReturnValue(w,1000,"","delete all jobs ok.")
}
func (ctrl *Controller) CancelSendCurrentEmails(w http.ResponseWriter,req *http.Request) {
	if req.Method != "DELETE"  {
		ctrl.ReturnValue(w,1300,"","invalid method " + req.Method)
		return 
	}
	if  ctrl.SmtpEnabled == true {
		for _,item := range ctrl.SendMailJobs.Members() {
			ctrl.SendMailJobs.Remove(item)
		}
	}
	ctrl.ReturnValue(w,1000,"","cancel operation of send current jobs succeed.")
}
