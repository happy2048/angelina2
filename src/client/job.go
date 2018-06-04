package client
import(
	"net/http"
	"fmt"
	"os"
	"time"
	"myutils"
	"strings"
	"io/ioutil"
	"encoding/json"
)
type ReturnData struct {
	Code int  `json:"code"`
	Msg  string  `json:"msg"`
	Data string   `json:"data"`
}
func Operate(method,url,data string) (string,error) {
    client := &http.Client{}
	var request *http.Request
    var err error
    if data == "" {
        request,err = http.NewRequest(method,url,nil)
    }else {
        request,err = http.NewRequest(method,url,strings.NewReader(data))
    }
	request.Header.Set("Connection", "keep-alive")
	response,err := client.Do(request)
	if err != nil {
		return "",err
	}
	if response.StatusCode == 200 {
		body,err := ioutil.ReadAll(response.Body)
		if err != nil {
		 return "",err
		}
		return string(body),nil
	}
	return "",fmt.Errorf("%s","requst failure")
}

func (cc *Connector) CancelSendEmails() {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/cancelEmails`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("DELETE",url,"")
	if err != nil {
		pstr := fmt.Sprintf("cancel sending emails failed,reason: %s",err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	myutils.Print("Info",data.Msg,false)
	os.Exit(0)
}
func (cc *Connector) CheckTempIsExist(name string) bool {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/checkTemp?name=%s`
	url = fmt.Sprintf(url,host,name)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("check template %s failed,reason: %s",name,err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	if data.Data == "" {
		myutils.Print("Info",data.Msg,true)
	}
	if data.Data == "true" {
		return true
	}else {
		return false
	}
}
func (cc *Connector) DelJobs() {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/cancelJobs`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("DELETE",url,"")
	if err != nil {
		pstr := fmt.Sprintf("delete all jobs failed,reason: %s",err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	myutils.Print("Info",data.Msg,false)
	os.Exit(0)
}
func (cc *Connector) StoreTemplate(info string) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/storeTemp`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("POST",url,info)
	if err != nil {
		pstr := fmt.Sprintf("store template failed,reason: %s",err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	myutils.Print("Info",data.Msg,false)
	os.Exit(0)
} 
func (cc *Connector) DeleteTemplate(name string) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/deleteTemp?name=%s`
	url = fmt.Sprintf(url,host,name)
	redata,err := Operate("DELETE",url,"")
	if err != nil {
		pstr := fmt.Sprintf("delete template %s failed,reason: %s",name,err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	myutils.Print("Info",data.Msg,false)
	os.Exit(0)
}
func (cc *Connector) GetTemplateContent(name string) string {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/getTemp?name=%s`
	url = fmt.Sprintf(url,host,name)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("query template %s failed,reason: %s",name,err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	return data.Data
}
func (cc *Connector) GetTemplateCon(name string) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/getTemp?name=%s`
	url = fmt.Sprintf(url,host,name)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("query template %s failed,reason: %s",name,err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	if data.Data != "" {
		fmt.Println(data.Data)
		os.Exit(0)
	}
	myutils.Print("Info",data.Msg,true)
}
func (cc *Connector) GetAllTemplates() {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/queryTemp`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("query templates failed,reason: %s",err.Error())
		myutils.Print("Error",pstr,true)
	}
	var data ReturnData
	err = json.Unmarshal([]byte(redata),&data)
	if err != nil {
		myutils.Print("Error","parse return message failed,exit.",true)
	}
	if data.Data != "" {
		fmt.Println(data.Data)
		os.Exit(0)
	}
	myutils.Print("Info",data.Msg,true)
}
func (cc *Connector) DeleteJob(job string) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/job?job=%s&operate=delete`
	url = fmt.Sprintf(url,host,job)
	redata,err := Operate("POST",url,"")
	if err != nil {
		pstr := fmt.Sprintf("delete job %s failed,reason: %s",job,err.Error())
		myutils.Print("Error",pstr,true)
	}
	if redata != "" {
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			myutils.Print("Error","parse return message failed,exit.",true)
		}
		myutils.Print("Info",data.Msg,false)
		os.Exit(0)	
	}
}
func (cc *Connector) GetAllJobStatus(nice bool) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/query`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("query all jobs' status failed,reason: %s",err.Error())
		myutils.Print("Error",pstr,true)
	}
	if redata != "" {
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			pstr := fmt.Sprintf("parse return message failed,reason: %s",err.Error())
			myutils.Print("Error",pstr,true)
		}
		if data.Data == "" {
			myutils.Print("Info","no jobs in the angelina.",false)
			os.Exit(0)
		}
		if !nice {
			fmt.Println(data.Data)
		}else {
			for _,line := range strings.Split(data.Data,"\n") {
				fmt.Println(line)
				time.Sleep(700 * time.Millisecond)
			}
		} 
	}
}
func (cc *Connector) GetJobStatus(job string,nice bool) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/job?job=%s&operate=status`
	url = fmt.Sprintf(url,host,job)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("get job %s status failed,reason: %s",job,err.Error())
		myutils.Print("Error",pstr,true)
	}
	if redata != "" {
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			pstr := fmt.Sprintf("parse return message failed,reason: %s",err.Error())
			myutils.Print("Error",pstr,true)
		}
		if data.Data == "" {
			pstr := fmt.Sprintf("get the status of job %s is null",job)
			myutils.Print("Info",pstr,true)
		}
		if !nice {
			fmt.Println(data.Data)

		}else {
			tdata := strings.Split(data.Data,"\n")
			for _,line := range tdata {
				fmt.Println(line)
				time.Sleep(700 * time.Millisecond)
			}
		}
	}

}
func (cc *Connector) RoundGetAllJobStatus() {
	myticker := time.NewTicker(8 * time.Second)
	cc.GetAllJobStatus(true)
	for {
		select {
			case <- myticker.C:
				cc.GetAllJobStatus(true)
		}
	}

}
func (cc *Connector) RoundGetJobStatus(job string) {
	myticker := time.NewTicker(8 * time.Second)
	cc.GetJobStatus(job,true)
	for {
		select {
			case <- myticker.C:
				cc.GetJobStatus(job,true)
		}
	}
}
