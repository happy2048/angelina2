package client
import (
	"fmt"
	"sync"
	"validator"
	"os"
	"cpfile"
	"encoding/json"
	"myutils"
	"strings"
	"path"
)
type BackJobs struct {
    Waiting []string `json:"waiting"`
    Running []string `json:"running"`
    Starting []string `json:"starting"`
}
type BatchClient struct {
	Names []string
	Inputs []string
	BaseDir string
	TmpTemplate string
	TemplateName string
	Template string
	Cover string
	ControllerAddr string
	JobRunningKey string
	ClientArray []*Client
}
type Client struct {
	Sample string
	Prefix string
	InputDir string
	PipeObj   *validator.Validator  
}
func NewBatchClient(conAddr,glusterDir,template,cover,tname,istmp string,params map[string]string,names,inputs []string) *BatchClient {
	referDir := "/mnt/refer"
	dataDir := "/mnt/data"
	clis := make([]*Client,0,len(names))
	for ind,name := range names {
		cli := NewClient(template,name,inputs[ind],referDir,dataDir,params)
		clis = append(clis,cli)
	}
	return &BatchClient {
		Inputs: inputs,
		Names: names,
		JobRunningKey: "angelina-running-jobs",
		TemplateName: tname,
		TmpTemplate: istmp,
		Template: template,
		ControllerAddr: conAddr,
		BaseDir: glusterDir,
		Cover: cover,
		ClientArray: clis}

}
func NewClient(template,name,indir,referDir,dataDir string,params map[string]string) *Client {
	prefix := myutils.GetSamplePrefix(name)
	va,err := validator.NewValidator(template,referDir,path.Join(dataDir,name),params)
	if err != nil {
		myutils.Print("Error",err.Error(),true)
	}
	va.StartValidate()
	return &Client {
		Prefix: prefix,
		Sample: name,
		InputDir: indir,
		PipeObj: va}
}
func (bcli *BatchClient) Start() {
	var wg sync.WaitGroup
	for ind,name := range bcli.Names {
		status := bcli.CheckSampleIsRunning(name)
		if status == "Running" || status == "Starting" || status == "WaitToRun" {
			pstr := fmt.Sprintf("job %s is %s,we don't init it",name,status)
			myutils.Print("Info",pstr,false)
			continue
		}
		wg.Add(1)
		go func(tname string,index int) {
			defer wg.Done()
			bcli.Init(tname,index)
		}(name,ind)
	}
	wg.Wait()
}
func (bcli *BatchClient) CheckSampleIsRunning(name string) string {
	host := strings.Trim(bcli.ControllerAddr," ")
	url := `http://%s/checkJob?name=%s`
	url = fmt.Sprintf(url,host,name)
	redata,err := Operate("GET",url,"")
	if err != nil {
		pstr := fmt.Sprintf("check job status %s failed,reason: %s",name,err.Error())
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
	return data.Data
}
func (bcli *BatchClient) Init(name string, index int) {
	bcli.CopyFile(name,index)
	bcli.RunAllStepsAgain(name,index)
	bcli.CreateJob(name)
}
func (bcli *BatchClient) CreateJob(name string) {
	host := strings.Trim(bcli.ControllerAddr," ")
	job := strings.Trim(name," ")
	url := `http://%s/job?job=%s&operate=create`
	url = fmt.Sprintf(url,host,job)
	redata,err := Operate("GET",url,"")
	if err != nil {
		fmt.Printf("create job %s failed,reason: %s\n",job,err.Error())
		return
	}
	if redata != "" {
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			fmt.Printf("parse return message failed,exit\n")
			return
		}
		myutils.Print("Info",data.Msg,false)
	}
}
func (bcli *BatchClient) CopyFile(name string,index int) {
	os.MkdirAll(path.Join(bcli.BaseDir,name,"step0"),0755)
	for key,_ := range bcli.ClientArray[index].PipeObj.NormData {
		os.MkdirAll(path.Join(bcli.BaseDir,name,key,"logs"),0755)
	}
	cpfile.CopyFilesToGluster(path.Join(bcli.BaseDir,name,"step0"),bcli.Inputs[index],bcli.Template)
	os.Remove(path.Join(bcli.BaseDir,name,"step0",".template"))
	bcli.ClientArray[index].PipeObj.WriteObjToFile(path.Join(bcli.BaseDir,name,"step0","pipeline.json"))
	if bcli.TmpTemplate == "false" {
		myutils.WriteFile(path.Join(bcli.BaseDir,name,"step0",".template"),bcli.TemplateName,true)
	}
}
func (bcli *BatchClient) RunAllStepsAgain(name string,index int) {
	if bcli.Cover == "true" {
		for key,_ := range bcli.ClientArray[index].PipeObj.NormData {
			os.Remove(path.Join(bcli.BaseDir,name,key,".status"))
		}
	}

}
