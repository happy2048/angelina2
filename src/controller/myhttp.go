package controller
import (
    "io"
	"io/ioutil"
    "net/http"
	"fmt"
	"strings"
	"myutils"
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
func (ctrl *Controller) HttpServer() {
	http.HandleFunc("/job",ctrl.JobsOptionServe)
	http.HandleFunc("/query",ctrl.QueryOptionServe)
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
		ReturnValue(w,1000,data,"get data ok")
	}else {
		ReturnValue(w,1000,"","invalid url method ")
	}
}
func (ctrl *Controller) HandleRequests(w http.ResponseWriter,req *http.Request) {
	if req.Method == "GET" || req.Method == "POST" {
		req.ParseForm()
		if len(req.Form["job"]) != 1 {
			ReturnValue(w,1101,"","invalid url,no job given or job gives more than one.")
			return 
		}
		if len(req.Form["operate"]) != 1 {
			ReturnValue(w,1102,"","invalid url,no option given or option gives more than one.")
			return 
		}
		if req.Method == "POST" && req.Form["operate"][0] == "delete"  {
			name := strings.Trim(req.Form["job"][0]," ")
			if strings.Index(name,"pipe") == 0 {
				job := ctrl.NameMap.Read(name)
				if job == "" {
					ReturnValue(w,1000,"","invalid job id " + name )
					return	
				} 
				name = job
			}
			ctrl.WaitingDeleteJobs.Add(name)
			ReturnValue(w,1000,"",name + " will be deleted.")
			return 
		}  
		if req.Method == "GET" && req.Form["operate"][0] == "create" {
			name := strings.Trim(req.Form["job"][0]," ")
			ReturnValue(w,1000,"",name + " has received.")
			id := myutils.GetSamplePrefix(name)
			ctrl.NameMap.Add(id,name)
			ctrl.AppendLogToQueue("Info","get message to create job ",name)
			ctrl.WaitingRunJobs.Add(name)
			return
		}
		if req.Method == "GET" && req.Form["operate"][0] == "status" {
			data := ctrl.GetJobStatus(req.Form["job"][0])
			ReturnValue(w,1000,data,"get status ok")
			return 
		}
		ReturnValue(w,1103,"","invalid url request.")
		 
	}else {
		msg := fmt.Sprintf(`invalid method %s for path %s`,req.Method,req.URL.Path)
		ReturnValue(w,1100,"",msg)
	}
}

func ReturnValue(w http.ResponseWriter,code int,redata,msg string) {
	format := `{"code": %d,"msg":"%s","data": "%s"}`
	data := fmt.Sprintf(format,code,msg,redata)
	io.WriteString(w,data)
}

