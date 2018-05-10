package client
import(
	"fmt"
	"io/ioutil"
	"myutils"
	"strings"
	"strconv"
	"bufio"
	"os"
	"validator"
	gjson "github.com/tidwall/gjson"
)

func (cc *Connector) StorePipeline() {
	if cc.Opt.Editor.PushTemp == "" {
		return 
	}
	var pname string
	var pdesc string
	var pcon  string
	var read  string
	bdata,err := ioutil.ReadFile(cc.Opt.Editor.PushTemp)
	if err != nil {
		fmt.Printf("Error: read template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	data := string(bdata)
	if ! gjson.Valid(data) {
		fmt.Printf("invalid pipeline file,parse failed,some commas are add in bad area or don't delete the annotation?\n")
		os.Exit(3)
	}
	jsonObj := gjson.Parse(data)
	jsonObj.ForEach(func(key,value gjson.Result)bool{
		if key.String() == "pipeline-name" {
			pname = strings.Trim(value.String()," ")
			if pname == "" {
				fmt.Println("Error: the field \"pipeline-name\" of pipeline template file is null")
				os.Exit(3)
			} 
		}else if key.String() == "pipeline-description" {
			pdesc = value.String()
		}else if key.String() == "pipeline-content" {
			pcon = strings.Trim(value.String()," ")
			if pcon == "" {
				fmt.Println("Error: the field \"pipeline-content\" of pipeline template file is null")
				os.Exit(3)
			}
		}
		return true
	})
	
	if cc.CheckPipelineExist(pname) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("the pipeline has been exists,do you want to cover it?[yes/no]: ")
		rdata,_,_ := reader.ReadLine()
		read = string(rdata)
		if read == "yes" {
			cc.CheckAndStorePipe(pname,pdesc,pcon)
			os.Exit(0)
		}else if read == "no" {
			os.Exit(0)
		}else {
			fmt.Printf("Error: you only can input \"yes\" or \"no\"\n")
			os.Exit(3)
		}
	}	
	cc.CheckAndStorePipe(pname,pdesc,pcon)
	os.Exit(0)
}
func (cc *Connector) CheckAndStorePipe(name,desc,con string) {
	va,err := validator.NewValidator(con,"/tmp","/tmp",make(map[string]string))
	if err != nil {
		fmt.Printf("Error: validate the pipeline template file failed,reason:%s \n",err.Error())
		os.Exit(3)
	}
	va.StartValidate()
	va.WriteObjToFile("/tmp/validate.json")
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	_,err = cc.Db.RedisSetAdd(redisKey,pipeid)
	if err != nil {
		fmt.Printf("Error: store pipeline template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	_,err = cc.Db.RedisHashSet(pipeid,"pipeline-name",name)
	if err != nil {
		fmt.Printf("Error: store pipeline template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	} 
	_,err = cc.Db.RedisHashSet(pipeid,"pipeline-description",desc)
	if err != nil {
		fmt.Printf("Error: store pipeline template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	_,err = cc.Db.RedisHashSet(pipeid,"pipeline-content",con)
	if err != nil {
		fmt.Printf("Error: store pipeline template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	_,err = cc.Db.RedisHashSet(pipeid,"estimate-time","0")
	if err != nil {
		fmt.Printf("Error: store pipeline template file failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
}
func (cc *Connector) ListAllTemp() {
	if cc.Opt.Editor.DisplayTemp == false {
		return 
	}
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	members,err := cc.Db.RedisSetMembers(redisKey)
	if err != nil {
		fmt.Printf("Error: list pipelines failed,reason: %s\n",err.Error())
		os.Exit(3)
	}
	fmt.Printf("%s\t%s\t%s\t%s\n",NormString("Pipeline Id",21),NormString("Pipeline Name",25),NormString("Estimate Time",20),"Pipeline Description")
	for _,pid := range members {
		name,err := cc.Db.RedisHashGet(pid,"pipeline-name")
		if err != nil {
			fmt.Printf("Error: get pipeline %s failed,reason: %s\n",pid,err.Error())
			continue
		}
		desc,err := cc.Db.RedisHashGet(pid,"pipeline-description")
		if err != nil {
			fmt.Printf("Error: get pipeline %s failed,reason: %s\n",pid,err.Error())
			continue
		}
		tm,err := cc.Db.RedisHashGet(pid,"estimate-time")
		if err != nil {
			fmt.Printf("Error: get pipeline %s failed,reason: %s\n",pid,err.Error())
			continue
		}
		tint,err := strconv.ParseInt(tm,10,64)
		if err != nil {
			fmt.Printf("Error: get pipeline %s failed,reason: %s\n",pid,err.Error())
			continue
		}
		tmstr := myutils.GetRunTimeWithSeconds(tint)
		fmt.Printf("%s\t%s\t%s\t%s\n",NormString(pid,21),NormString(name,25),NormString(tmstr,20),desc)
		
	}
	os.Exit(0)
}
func (cc *Connector) DeletePipeline() {
	if cc.Opt.Editor.DeleteTemp == "" {
		return 
	}
	name := cc.Opt.Editor.DeleteTemp
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	cc.Db.RedisSetSremMember(redisKey,pipeid)
	cc.Db.RedisDelKey(pipeid)
	os.Exit(0)
}
func (cc *Connector) DisplayPipeline() {
	if cc.Opt.Editor.QueryTemp == "" {
		return 
	} 
	info := cc.Opt.Editor.QueryTemp
	data,err := cc.GetPipelineContent(info)
	if err == nil {
		fmt.Println(data)
		os.Exit(0)
	}
	data,err = cc.Db.RedisHashGet(info,"pipeline-content")
	if err != nil {
		fmt.Printf("Error: get %s content failed,reason: %s\n",info,err.Error())
		os.Exit(3)
	}
	fmt.Println(data)
	os.Exit(0)
}
func (cc *Connector) GetPipelineContent(name string) (string,error) {
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	data,err := cc.Db.RedisHashGet(pipeid,"pipeline-content")
	if  err == nil {
		return data,nil
	}
	data,err = cc.Db.RedisHashGet(strings.Trim(name," "),"pipeline-content")
	return data,err
}
func (cc *Connector) CheckPipelineExist(name string) bool {
	redisKey := "pipeline" + myutils.GetSha256("pipeline")[:20]
	pipeid := "pipeid" + myutils.GetSha256(strings.Trim(name," "))[:15]
	status,err := cc.Db.RedisSetSisMember(redisKey,pipeid)
	if err != nil {
		fmt.Printf("Error: access redis failed,reason: %s\n",err.Error())
		os.Exit(2)
	}
	return status
}
func NormString(info string,length int) string {
    llen := len(info)
    if llen <= length {
        return info + strings.Repeat(" ",length - llen)
    }else {
        return info[0:length]
    }

}
