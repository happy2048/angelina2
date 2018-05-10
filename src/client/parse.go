package client
import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"redisdb"
	"myutils"
	"strings"
	gjson "github.com/tidwall/gjson"
	"io/ioutil"
)
type ReturnValue struct {
	RedisAddr 		string
	Sample    		string
	ControllerAddr  string
	PipeTemp  		string
	PipeTempName    string
	GlusterEntryDir string
	Force 			string
	Input		 	string
	Tmp             string
	Env 			map[string]string
}
type Connector struct {
	Opt *Options
	Rv  *ReturnValue
	Db  redisdb.Database
}

type EditorOptions struct {
	//Init   string `short:"I" long:"init" description:"Angelina configure file,the content of the file will be stored in the redis,and \n use -g option will generate an angelina template configure file."`
	PushTemp  string `short:"s" long:"store" description:"Give a pipeline template file,and store it to redis."`
	DisplayTemp  bool `short:"l" long:"list" description:"List the pipelines which have already existed."`
	DeleteTemp   string `short:"D" long:"delete" description:"Delete the pipeline." default:""`
	DeleteJob    string `short:"d" long:"del" description:"Given the job id or job name,Delete the job" default:""`
	Job          string `short:"j" long:"job" description:"Given the job id or job name,get the job status." default:""`
	AllJobStatus bool    `short:"J" long:"jobs" description:"Get  all jobs status"`
	Persist      bool `short:"k" long:"keeping" description:"Get the job status(or all jobs status) all the time,must along with -j or -J."`
	QueryTemp  string  `short:"q" long:"query" description:"give the pipeline id or pipeline name to get it's content." default:""`
	Gener   string  `short:"g" long:"generate" description:"Three value(\"conf\",\"pipe\") can be given,\"pipe\" is to generate a pipeline \n template file and you can edit it and use -s to store the pipeline;\"conf\" is to \n generate running configure file and you can edit it and use -c option to run the \n sample" default:""`
}

type Options struct {
	Version bool `short:"v" long:"version" description:"software version."`
	Force bool `short:"f" long:"force" description:"force to run all step of the sample,ignore they are succeed or failed last time."`
	Sample string `short:"n" long:"name" description:"Sample name." default:"no name" default:""`
	InputDir string `short:"i" long:"input" description:"Input directory,which includes some files  that are important to run the sample." default:""` 
	OutPutDir string `short:"o" long:"output" description:"Output directory,which is a glusterfs mount point,so that copy files to glusterfs." default:""`
	Template  string `short:"t" long:"template" description:"Pipeline template name,the sample will be running by the pipeline template." default:""`
	TmpTemp   string `short:"T" long:"tmp" description:"A temporary pipeline template file,defines the running steps,the sample will be \n running by it,can't be used with -t." default:""`
	Env   []string `short:"e" long:"env" description:"Pass variable to the pipeline template such as TEST=\"test\",this option can be \n used many time,eg: -e TEST=\"test1\" -e NAME=\"test\"."`
	Conf  string `short:"c" long:"config" description:"configure file,which include the values of -f -n -i -o -t."`
	Controller string `short:"a" long:"angelina" description:"Angelina Controller address like ip:port,if you don't set this option,you must set the System Environment Variable ANGELINA." default:""`
	Redis string `short:"r" long:"redis" description:"Redis server address,can't use localhost:6379 and 127.0.0.1:6379,because they can't \n be accessed by containers,give another address;if the -r option don't give,you must \n set the System Environment Variable REDISADDR." default:""`
	Editor EditorOptions `group:"Other Options"`

}
func NewConnector() *Connector {
	rv  := &ReturnValue {
		Input: "",
		Force: "false",
		Tmp: "false",
		PipeTemp: "",
		RedisAddr:"",
		Sample: "",
		PipeTempName: "",
		GlusterEntryDir: "",
		Env: make(map[string]string)}
	opt := NewOptions() 
	return &Connector{
		Rv: rv,
		Opt: opt}
}
func (cc *Connector) Start() {
	cc.Opt.Start()
	cc.CheckController()
	cc.GetJobsStatus()
	cc.PrintJobInfo()
	cc.DeleteMyJob()
	cc.CheckConfig()
	cc.CheckNoConfig()
	cc.CheckRedis()
	//cc.InitAngelina()
	cc.StorePipeline()
	cc.DeletePipeline()	
	cc.ListAllTemp()
	cc.DisplayPipeline()
	cc.LastCheck()
	cc.IsTmpTemplate()
}
func (cc *Connector) ReturnInfo() *ReturnValue {
	return cc.Rv
}
func (cc *Connector) DeleteMyJob() {
	if cc.Opt.Editor.DeleteJob == "" {
		return 
	}
	job := cc.Opt.Editor.DeleteJob
	cc.DeleteJob(job)
}
func (cc *Connector) PrintJobInfo() {
	if cc.Opt.Editor.Job == "" {
		return
	}
	job := cc.Opt.Editor.Job
	if cc.Opt.Editor.Persist {
		cc.RoundGetJobStatus(job)
	}
	cc.GetJobStatus(job,false)
	os.Exit(0)
}
func (cc *Connector) GetJobsStatus() {
	if cc.Opt.Editor.AllJobStatus  {
		if cc.Opt.Editor.Persist {
			cc.RoundGetAllJobStatus()
		}else {
			cc.GetAllJobStatus(false)
			os.Exit(0)
		}
	}
}
func (cc *Connector) CheckController() {
	if cc.Opt.Controller == "" {
		if myutils.GetOsEnv("ANGELINA") == "" {
			fmt.Println("Error: not set angelina controller address,we don't know how to connect it")
			os.Exit(2)
		}
		cc.Rv.ControllerAddr = myutils.GetOsEnv("ANGELINA")
	}else {
		cc.Rv.ControllerAddr = cc.Opt.Controller
	}
}
func (cc *Connector) Print() {
	data := `InputDir:%s
Force: %s
Tmp: %s
PipeTemp: %s
RedisAddr: %s
Sample: %s
GlusterEntryDir: %s
Env: %s\n`
fmt.Printf(data,cc.Rv.Input,cc.Rv.Tmp,cc.Rv.Force,cc.Rv.PipeTemp,cc.Rv.RedisAddr,cc.Rv.Sample,cc.Rv.GlusterEntryDir,cc.Rv.Env)
}
func (cc *Connector) IsTmpTemplate() {
	if cc.Opt.TmpTemp != "" {
		cc.Rv.Tmp = "true"
	}
}
func (cc *Connector) LastCheck() {
	if cc.Rv.Input == "" {
		fmt.Println("Error: input directory is null")
		os.Exit(3)
	}
	if cc.Rv.RedisAddr == "" {
		fmt.Println("Error: redis address is null")
		os.Exit(3)
	}
	if cc.Rv.Sample == "" {
		fmt.Println("Error: sample name is null")
		os.Exit(3)
	}
	if cc.Rv.GlusterEntryDir == "" {
		ge,err := cc.Db.RedisHashGet("kubernetesConfig","OutputBaseDir")
		if err != nil {
			fmt.Println("Error: gluster entry directory is null")
			os.Exit(3)
		}
		cc.Rv.GlusterEntryDir = ge
	}
	if cc.Rv.PipeTemp == "" {
		if cc.Rv.PipeTempName == "" {
			fmt.Println("Error: the pipeline template name is null")
			os.Exit(3)
		}else {
			con,err := cc.GetPipelineContent(cc.Rv.PipeTempName)
			if err != nil {
				fmt.Printf("Error: get pipline template \"%s\" failed,reason: %s\n",cc.Rv.PipeTempName,err.Error())
				os.Exit(3)
			}
			if con == "" {
				fmt.Printf("Error: get pipeline template failed,reason: pipeline does not exist.\n")
				os.Exit(3)	
			}
			cc.Rv.PipeTemp = con
		}
	}else {
		if ! gjson.Valid(cc.Rv.PipeTemp) {
			fmt.Printf("invalid pipeline file,parse failed,some commas are add in bad area or don't delete the annotation?\n")
			os.Exit(3)
		}
		jsonObj := gjson.Parse(cc.Rv.PipeTemp)
		jsonObj.ForEach(func(key,value gjson.Result)bool{
			if key.String() == "pipeline-content" {
				pcon := strings.Trim(value.String()," ")
				if pcon == "" {
					fmt.Println("Error: the field \"pipeline-content\" of pipeline template file is null")
					os.Exit(3)
				}
				cc.Rv.PipeTemp = pcon
			}
			return true
		})
	}
}
/*
func (cc *Connector) InitAngelina() {
	if cc.Opt.Editor.Init == "" {
		return 
	}
	data,err := ioutil.ReadFile(cc.Opt.Editor.Init)
	if err != nil {
		fmt.Printf("Error: read file %s failed,reason: %s\n",cc.Opt.Editor.Init,err.Error())
		os.Exit(3)
	}
	cc.ReadConfig(string(data))
	os.Exit(0)
}
*/
func (cc *Connector) CheckRedis() {
	cc.Rv.RedisAddr = strings.Trim(cc.Opt.Redis," ")
	if cc.Rv.RedisAddr == "" {
		redisAddr := myutils.GetOsEnv("REDISADDR")
		if redisAddr == "" {
			fmt.Println("Error: no redis server give,you can use -r option or set System Environment Variable REDISADDR to assign it.")
			os.Exit(2)
		}else {
			cc.Rv.RedisAddr = redisAddr
		}	
	}
	if strings.Index(cc.Rv.RedisAddr,"127.0.0.1") != -1 {
		fmt.Printf("Error: we check that you set redis address with 127.0.0.1,it can't be used by container.\n")
		os.Exit(3)
	}  
	if strings.Index(cc.Rv.RedisAddr,"localhost") != -1 {
		fmt.Printf("Error: we check that you set redis address with localhost,it can't be used by container.\n")
		os.Exit(3)
	}  
	testErr := redisdb.RedisTestConnect(cc.Rv.RedisAddr)
	if testErr != nil {
		fmt.Printf("Error: connect redis failed,reason: %s\n",testErr.Error())
		os.Exit(3)
	}
	cc.Db = redisdb.NewRedisDB("tcp",cc.Rv.RedisAddr)
}

func (cc *Connector) CheckNoConfig() {
	if strings.Trim(cc.Opt.Sample," ") != "" {
		cc.Rv.Sample = strings.Trim(cc.Opt.Sample," ")
	}
	if strings.Trim(cc.Opt.InputDir," ") != "" {
		cc.Rv.Input  = strings.Trim(cc.Opt.InputDir," ")
	}
	if strings.Trim(cc.Opt.OutPutDir," ") != "" {
		cc.Rv.GlusterEntryDir = strings.Trim(cc.Opt.OutPutDir," ")
	}
	if strings.Trim(cc.Opt.Template," ") != "" {
		cc.Rv.PipeTempName = strings.Trim(cc.Opt.Template," ")
	}
	if cc.Opt.Force {
		cc.Rv.Force = "true"
	}
	if len(cc.Opt.Env) != 0 {
		for _,val := range cc.Opt.Env {
			val = strings.Trim(val," ")
			info := strings.Split(val,"=")
			if len(info) < 2 {
				fmt.Printf("Error: invalid value %s of -e option\n",val)
				os.Exit(3)
			}else {
				ename := info[0]
				evalue := strings.Join(info[1:],"=")
				cc.Rv.Env[ename] = evalue
			}
		}
	}
	if cc.Opt.TmpTemp != "" {
		data,err := ioutil.ReadFile(cc.Opt.TmpTemp)
		if err != nil {
			fmt.Printf("Error: read file %s failed,reason:%s\n",cc.Opt.TmpTemp,err.Error())
			os.Exit(3)
		}
		cc.Rv.PipeTemp = string(data)
	}
}
func (cc *Connector) CheckConfig() {
	if cc.Opt.Conf == "" {
		return 
	}
	conf := cc.Opt.Conf
	bdata,err := ioutil.ReadFile(conf)
	if err != nil {
		fmt.Printf("Error: read configure file error,reason: %s\n",err.Error())
		os.Exit(3)
	}
	data := string(bdata)
	if ! gjson.Valid(data) {
		fmt.Printf("Error: invalid configure file %s,parse failed,some commas are add in bad area or don't delete the annotation?\n",conf)
		os.Exit(3)
    } 
	jsonObj := gjson.Parse(data)
	jsonObj.ForEach(func(key,value gjson.Result)bool{
		if key.String() == "input-directory" {
			val := strings.Trim(value.String()," ")
			if val != "" {
				cc.Rv.Input = val
			}
		}else if key.String() == "glusterfs-entry-directory" {
			val := strings.Trim(value.String()," ")
			if val != "" {
				cc.Rv.GlusterEntryDir = val
			}

		}else if key.String() == "sample-name" {
			val := strings.Trim(value.String()," ")
			if val != "" {
				cc.Rv.Sample = val
			}

		}else if key.String() == "redis-address" {
			val := strings.Trim(value.String()," ")
			if val != "" {
				cc.Rv.RedisAddr = val
			}

		}else if key.String() == "template-env" {
			if len(value.Array()) != 0 {
				for _,ival := range value.Array() {
					val := strings.Trim(ival.String()," ")
					if val != "" {
						info := strings.Split(val,"=")
						if len(info) < 2 {
							fmt.Printf("Error: invalid string %s in template-env of configure file %s\n",val,conf)
							os.Exit(3)
						}else {
							ename := info[0]
							evalue := strings.Join(info[1:],"=")
							cc.Rv.Env[ename] = evalue
						}

					}
				}
			}
		}else if key.String() == "pipeline-template-name" {
			val := strings.Trim(value.String()," ")
			if val != "" {
                cc.Rv.PipeTempName = val
            }
		}else if key.String() == "force-to-cover" {
			val := strings.Trim(value.String()," ")
			if val == "yes" {
                cc.Rv.Force = "true"
            }else if val == "no" {
                cc.Rv.Force = "false"

			}else {
				fmt.Printf("Error: invalid value of force-to-cover in configure file %s,value must be \"yes\" or \"no\"\n",conf)
				os.Exit(3)
			}
		}
		return true
	})	

}
func NewOptions() (*Options) {
	var options Options
	_,err := flags.NewParser(&options, flags.Default).Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error);ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
	return &options
}
func (opt *Options) Start() {
	opt.Check()
	opt.CheckVersion()
	opt.GenerateTemp()
}
	
func (opt *Options) GenerateTemp() {
	if opt.Editor.Gener == "" {
		return
	}
	data := strings.Split(opt.Editor.Gener,",") 
	if len(data) !=0 {
		for _,val := range data {
			if strings.Trim(val," ") == "init" {
				ioutil.WriteFile("/tmp/angelina.json",[]byte(InitTemplate),0644)
        		fmt.Println("create init template file to /tmp/angelina.json")
			}else if strings.Trim(val," ") == "conf" {
				ioutil.WriteFile("/tmp/config.json",[]byte(ConfigTemplate),0644)
				fmt.Println("create configure template file to /tmp/config.json")
			}else if strings.Trim(val," ") == "pipe" {
				ioutil.WriteFile("/tmp/pipeline.json",[]byte(PipelineTemplate),0644)
        		fmt.Println("create pipeline template file to /tmp/pipeline.json")
			}else {
				fmt.Printf("invalid value: %s,you can give value from \"conf\" or \"pipe\" or \"init\"\n",val)
				os.Exit(3)
			}
		}
		os.Exit(0)
	}
} 
func (opt *Options) Check() {
	if len(os.Args) == 1 {
		fmt.Printf("Error: you should give some options to run angelina,plese use -h or --help to get usage.\n")
		os.Exit(1)
	}
}
func (opt *Options) CheckVersion() {
	if opt.Version {
		fmt.Println("Angelina v2.0 linux/amd64")
		os.Exit(0)
	}

}
/*
func main() {
	options,_ := NewOptions()
	fmt.Println(*options)
	options.Check()
}
*/
