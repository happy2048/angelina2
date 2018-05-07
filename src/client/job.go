package client
import(
	"net/http"
	"fmt"
	"os"
	"time"
	"strings"
	"io/ioutil"
	"encoding/json"
)
type ReturnData struct {
	Code int  `json:"code"`
	Msg  string  `json:"msg"`
	Data string   `json:"data"`
}
func Operate(method,url string) (string,error) {
    client := &http.Client{}
	request, _ := http.NewRequest(method,url, nil)
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
func (cc *Connector) DeleteJob(job string) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/job?job=%s&operate=delete`
	url = fmt.Sprintf(url,host,job)
	redata,err := Operate("POST",url)
	fmt.Println(redata)
	if err != nil {
		fmt.Printf("delete job %s failed,reason: %s\n",job,err.Error())
		os.Exit(3)
	}
	if redata != "" {
		redata = strings.Replace(redata,"\n","-***-",-1)
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			fmt.Printf("parse return message failed,exit\n")
			os.Exit(3)
		}
		data.Data = strings.Replace(data.Data,"-***-","\n",-1)
		fmt.Println(data.Data)
		os.Exit(0)	
	}
}
func (cc *Connector) GetAllJobStatus(nice bool) {
	host := strings.Trim(cc.Rv.ControllerAddr," ")
	url := `http://%s/query`
	url = fmt.Sprintf(url,host)
	redata,err := Operate("GET",url)
	if err != nil {
		fmt.Printf("query all jobs' status failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	if redata != "" {
		redata = strings.Replace(redata,"\n","-***-",-1)
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			fmt.Printf("parse return message failed,reason: %s\n",err.Error())
			os.Exit(3)
		}
		if data.Data == "" {
			fmt.Println("no jobs in the angelina")
			os.Exit(0)
		}
		data.Data = strings.Replace(data.Data,"-***-","\n",-1)
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
	redata,err := Operate("GET",url)
	if err != nil {
		fmt.Printf("get job %s status failed,reason: %s\n",job,err.Error())
		os.Exit(3)
	}
	if redata != "" {
		var data ReturnData
		redata = strings.Replace(redata,"\n","-***-",-1)
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			fmt.Printf("parse return message failed,reason: %s\n",err.Error())
			os.Exit(3)
		}
		if data.Data == "" {
			fmt.Printf("get the status of job %s is null\n",job)
		}
		if !nice {
			data.Data = strings.Replace(data.Data,"-***-","\n",-1)
			fmt.Println(data.Data)

		}else {
			data.Data = strings.Replace(data.Data,"-***-","\n",-1)
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
