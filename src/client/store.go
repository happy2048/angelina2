package client
import(
	"fmt"
	"io/ioutil"
	"strings"
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
	if cc.CheckTempIsExist(pname) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("the pipeline has been exists,do you want to cover it?[yes/no]: ")
		rdata,_,_ := reader.ReadLine()
		read = string(rdata)
		if read == "no" {
			os.Exit(0)
		}else if read != "yes" {
			fmt.Printf("Error: you only can input \"yes\" or \"no\"\n")
			os.Exit(3)
		}
	}
	va,err := validator.NewValidator(pcon,"/tmp","/tmp",make(map[string]string))	
	if err != nil {
		fmt.Printf("Error: validate the pipeline template file failed,reason:%s \n",err.Error())
		os.Exit(3)
	}
	va.StartValidate()
	va.WriteObjToFile("/tmp/validate.json")
	cc.StoreTemplate(data)
	os.Exit(0)
}
func (cc *Connector) ListAllTemp() {
	if cc.Opt.Editor.DisplayTemp == false {
		return 
	}
	cc.GetAllTemplates()
}
func (cc *Connector) DeletePipeline() {
	if cc.Opt.Editor.DeleteTemp == "" {
		return 
	}
	name := cc.Opt.Editor.DeleteTemp
	cc.DeleteTemplate(name)
}
func (cc *Connector) DisplayPipeline() {
	if cc.Opt.Editor.QueryTemp == "" {
		return 
	} 
	info := cc.Opt.Editor.QueryTemp
	cc.GetTemplateCon(info)
}
func (cc *Connector) CancelAllEmail() {
	if cc.Opt.Editor.CancelSendEmail == false {
		return 
	}
	cc.CancelSendEmails()
}
func (cc *Connector) DeleteAllJobs() {
	if cc.Opt.Editor.DeleteAllJobs == false {
		return 
	}
	cc.DelJobs()
}
func NormString(info string,length int) string {
    llen := len(info)
    if llen <= length {
        return info + strings.Repeat(" ",length - llen)
    }else {
        return info[0:length]
    }

}
