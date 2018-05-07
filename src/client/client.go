package client
import (
	"fmt"
	"redisdb"
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
type Client struct {
	Db redisdb.Database
	Sample string
	Prefix string
	InputDir string
	BaseDir string
	TmpTemplate string
	TemplateName string
	Template string
	Cover string
	ControllerAddr string
	JobRunningKey string
	PipeObj   *validator.Validator  
}
/*
func main() {
	data,_ := ioutil.ReadFile("pipelineTest.json")
	cli := NewClient("10.61.0.86:6379","yang2","/mnt/data","yang1",string(data),"true",make(map[string]string))
	cli.Start()
}
*/
func NewClient(conAddr,redisAddr,inputDir,glusterDir,sample,template,cover,tname,istmp string,params map[string]string) *Client {
	db := redisdb.NewRedisDB("tcp",redisAddr)
	prefix := myutils.GetSamplePrefix(sample)
	indir := inputDir
	referDir := "/mnt/refer"
	dataDir := "/mnt/data"
	va,err := validator.NewValidator(template,referDir,path.Join(dataDir,sample),params)
	if err != nil {
		myutils.Print("Error",err.Error(),true)
	}
	va.StartValidate()
	return &Client {
		Db: db,
		JobRunningKey: "angelina-running-jobs",
		TemplateName: tname,
		TmpTemplate: istmp,
		Sample: sample,
		Prefix: prefix,
		InputDir: indir,
		ControllerAddr: conAddr,
		BaseDir: path.Join(glusterDir,sample),
		Cover: cover,
		Template: template,
		PipeObj: va}
}
func (cli *Client) Start() {
	if cli.CheckSampleIsRunning() == true {
		fmt.Println("the sample is running,exit")
		os.Exit(2)
	}else {
		cli.Init()
	}
}
func (cli *Client) CheckSampleIsRunning() bool{
    var back BackJobs
	sample := cli.Sample
    data,err := cli.Db.RedisStringGet(cli.JobRunningKey)
    if err != nil {
        return false
    }
    err = json.Unmarshal([]byte(data),&back)
    if err != nil {
        return false
    }
    for _, job := range back.Running {
		if job == sample {
			return true
		}
    }
    for _, job := range back.Starting {
		if job == sample {
			return true
		}
		
    }
    for _, job := range back.Waiting {
		if job == sample {
			return true
		}
    }
	return false
}
func (cli *Client) Init() {
	cli.CopyFile()
	cli.RunAllStepsAgain()
	cli.CreateJob()
}
func (cli *Client) CreateJob() {
	host := strings.Trim(cli.ControllerAddr," ")
	job := strings.Trim(cli.Sample," ")
	url := `http://%s/job?job=%s&operate=create`
	url = fmt.Sprintf(url,host,job)
	redata,err := Operate("GET",url)
	if err != nil {
		fmt.Printf("create job %s failed,reason: %s\n",job,err.Error())
		os.Exit(3)
	}
	if redata != "" {
		var data ReturnData
		err := json.Unmarshal([]byte(redata),&data)
		if err != nil {
			fmt.Printf("parse return message failed,exit\n")
			os.Exit(3)
		}
		fmt.Println(data.Data)
		os.Exit(0)	
	}
}
func (cli *Client) CopyFile() {
	os.MkdirAll(path.Join(cli.BaseDir,"step0"),0755)
	for key,_ := range cli.PipeObj.NormData {
		os.MkdirAll(path.Join(cli.BaseDir,key,"logs"),0755)
	}
	cpfile.CopyFilesToGluster(path.Join(cli.BaseDir,"step0"),cli.InputDir,cli.Template)
	os.Remove(path.Join(cli.BaseDir,"step0",".template"))
	cli.PipeObj.WriteObjToFile(path.Join(cli.BaseDir,"step0","pipeline.json"))
	if cli.TmpTemplate == "false" {
		myutils.WriteFile(path.Join(cli.BaseDir,"step0",".template"),cli.TemplateName,true)
	}
}
func (cli *Client) RunAllStepsAgain() {
	if cli.Cover == "true" {
		for key,_ := range cli.PipeObj.NormData {
			os.Remove(path.Join(cli.BaseDir,key,".status"))
		}
	}

}
