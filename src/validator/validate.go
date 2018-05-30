package validator
import(
    "strings"
    "fmt"
	"os"
	"path"
    "strconv"
    "myutils"
    "regexp"
 	"encoding/json"
    gjson "github.com/tidwall/gjson"
)
type Step struct {
    Command string `json:command`
	TmpArgs []string `json:"-"`
	CommandName string `json:commandName`
    Args string `json:args`
    Container string `json:container`
    Prestep []string `json:prestep`
    SubArgs []string `json:subArgs`
	CmdArray []string `json:"-"`
	CommandCount string `json:commandCount`
	ResourcesLimits []string `json:resourcesLimits`
	ResourcesRequests []string `json:resourcesRequests`
}
type Validator struct {
	Data map[string]gjson.Result
	Refer map[string]string
	Param map[string]string
	ResourcesLimits map[string][]string
	ResourcesRequests map[string][]string
	Input []string
	NormData map[string]*Step
	BaseDir string
}
/*
func main() {
	data,_ := ioutil.ReadFile("./pipelineTest.json")
	redata,err := NewValidator(string(data),"/mnt/refer","/mnt/data",make(map[string]string))
	redata.StartValidate()
	redata.WriteObjToFile("/tmp/test.json")
	fmt.Println(redata.NormData)
	for _,val := range redata.NormData {
		fmt.Println(val)
	}
	fmt.Println(err)

}
*/
func GetStepIndex(data string) (int,error) {
    re := strings.Split(data,"step")
    if len(re) == 0 {
        return 0,fmt.Errorf("invalid string %s for parse index",data)
    }else if len(re) == 2 && re[0] == ""{
        redata,err := strconv.ParseInt(re[1],10,32)
        return int(redata),err
    }
    return 0,fmt.Errorf("invalid string %s for parse index",data)

}
func NormStep(data string) (string,error) {
	data = strings.Trim(data," ")
	ind,err := GetStepIndex(data)
	if err != nil {
		return "",err
	}
	redata := "step" + strconv.Itoa(ind)
	return redata,nil
}
func (vd *Validator) StartValidate() {
	vd.ValidateStepItem()
	vd.ValidateResourcesValue()
	vd.ValidateField()
	vd.SetTmpStepValue()
	vd.ValidatePreStep()
	vd.ReplaceFlag()
	vd.ValidateStepsSum()
	vd.JoinCommand()
}
func (vd *Validator) WriteObjToFile(file string) {
    json, err := json.Marshal(vd.NormData)
    if err != nil {
        myutils.Print("Error","Write pipeline to json failed",true)
    }
    data := strings.Replace(string(json),`\u003e`,`>`,-1)
    myutils.WriteFile(file,data,true)
}
func (vd *Validator) JoinCommand() {
	for key,value := range vd.NormData {
		cmd := strings.Join(value.CmdArray," ")
		tdata := strings.Join(value.TmpArgs," ")
		vd.NormData[key].Command = cmd
		vd.NormData[key].Args = tdata
	}
}
func (vd *Validator) ValidateStepsSum() {
	for i := 0;i < len(vd.NormData);i++ {
		if _,ok := vd.NormData["step" + strconv.Itoa(i+1)]; !ok {
			myutils.Print("Error","we checked that " + "step" + strconv.Itoa(i+1) + " not define,exit",true)
		}
	}

}
func (vd *Validator) ReplaceFlag() {
	regRefer := regexp.MustCompile(`refer@[\w|-|_|\.|*]+\b`)
	//regRefer := regexp.MustCompile(` \b[\w|-|_|\.|*]*refer@[\w|-|_|\.|*]+\b`)
	regParam := regexp.MustCompile(`params@[\w|-|_|\.|*]+\b`)
	//regParam := regexp.MustCompile(` \b[\w|-|_|\.|*]*params@[\w|-|_|\.|*]+\b`)
	stepRefer := regexp.MustCompile(`step[\d]+@`)
	data := vd.NormData
	for key,value := range data {
		tarr := value.TmpArgs
		oarr := value.SubArgs
		cmd := value.CmdArray
		vd.NormData[key].TmpArgs = ReplaceTarget(regRefer,"refer",vd.Refer,tarr)
		vd.NormData[key].SubArgs = ReplaceTarget(regRefer,"refer",vd.Refer,oarr)
		vd.NormData[key].CmdArray = ReplaceTarget(regParam,"params",vd.Param,cmd)
		vd.NormData[key].TmpArgs = ReplaceTarget(regParam,"params",vd.Param,tarr)
		vd.NormData[key].SubArgs = ReplaceTarget(regParam,"params",vd.Param,oarr)
		vd.NormData[key].TmpArgs = vd.ReplaceStepTarget(stepRefer,tarr)
		vd.NormData[key].SubArgs = vd.ReplaceStepTarget(stepRefer,oarr)
	}
}
func (vd *Validator) ReplaceStepTarget(regRefer  *regexp.Regexp,data []string) []string {
	for ind,val := range data {
    	tval := " " + val + " "
        result := regRefer.FindAllString(tval,-1)
        if len(result) != 0 {
        	for _,str := range result {
				str = strings.Trim(str," ")
				step := strings.Split(str,"@")[0]
				_,ok := vd.NormData[step]
				if strings.Trim(step," ") != "step0" && ! ok {
                 	myutils.Print("Warn","the " + step + " of string " + str + " not match the all steps",false)
				}else {
					all := strings.Split(str,"@")
					step := all[0]
					repstr := path.Join(vd.BaseDir,step)
					tval = strings.Replace(tval,step +"@",repstr + "/",-1)
				} 
           	}
			data[ind] = tval   
        }   
   	}
	return data
}
func ReplaceTarget(regRefer  *regexp.Regexp,flag string,refer map[string]string,data []string) []string {
	for ind,val := range data {
    	tval := " " + val + " "
        result := regRefer.FindAllString(tval,-1)
        if len(result) != 0 {
        	for _,str := range result {
				str = strings.Trim(str," ")
				//reflag := strings.Split(str,"@")[0]
				//prestr := strings.Split(reflag,flag)[0]
				tstr := strings.Join(strings.Split(str,"@")[1:],"@")
				if _,ok := refer[tstr];ok {
					tval = strings.Replace(tval,str,refer[tstr],-1)
              		//tval = regRefer.ReplaceAllString(tval, refer[tstr])
            	}else {
                 	myutils.Print("Warn","the string " + str + " can't match " + flag + " field",false)
                }   
           	}
			data[ind] = tval   
        }   
   	}
	return data
}
func (vd *Validator) ValidatePreStep() {
	for key,value := range vd.NormData {
		for _,ival := range value.Prestep {
			curIndex,_ := GetStepIndex(key)
			preIndex,err := GetStepIndex(strings.Trim(ival," "))	
			if err != nil {
				myutils.Print("Error","invalid prestep " + ival + " of " + key + ",reason: " + err.Error(),true)
			}
			prestep,_ := NormStep(ival)
			if _,ok := vd.NormData[prestep]; !ok {
				myutils.Print("Error","invalid prestep " + ival + " of " + key + ",because it is not defined",true)
			}
			if preIndex >= curIndex {
				myutils.Print("Error","invalid prestep " + ival + " of " + key + ",because it is behind " + key,true)
			}
		}

	}
}
func (vd *Validator) SetTmpStepValue() {
	data := vd.Data
	for key,value := range data {
		prestep := make([]string,0,len(value.Get("pre-steps").Array()))
		tmpargs := make([]string,0,len(value.Get("args").Array()))
		subargs := make([]string,0,len(value.Get("sub-args").Array()))
		command := make([]string,0,len(value.Get("command").Array()))
		cmdCount := "1"
		if cap(prestep) != 0 {
			for _,val := range value.Get("pre-steps").Array() {
				prestep = append(prestep,val.String())
			}
		}
		if cap(tmpargs) != 0 {
			for _,val := range value.Get("args").Array() {
				tmpargs =  append(tmpargs,val.String())
			}
		}
		if cap(subargs) != 0 {
			for _,val := range value.Get("sub-args").Array() {
				subargs = append(subargs,val.String())
			}
			cmdCount = strconv.Itoa(cap(subargs))
		}
		if cap(command) != 0 {
			for _,val := range value.Get("command").Array() {
				command = append(command,val.String())
			}
		}
		vd.NormData[key].SubArgs = subargs
		vd.NormData[key].TmpArgs = tmpargs
		vd.NormData[key].Prestep = prestep
		vd.NormData[key].CmdArray = command
		vd.NormData[key].CommandCount = cmdCount
	}

}
func (vd *Validator) ValidateField()  {
	data := vd.Data
	for key,value := range data {
		cmdName := value.Get("command-name").String()
		container := value.Get("container").String()
		limit := make([]string,2,2)
		request := make([]string,2,2)
		container = strings.Trim(container," ")
		if strings.Index(container,"[") == 0 {
			myutils.Print("Error","the field container of " + key + " is invalid,it should a string rather than array",true)
		}
		if container == "" {
			myutils.Print("Error","the field container of " + key + " is null",true)
		}
		cmdName = strings.Trim(cmdName," ")
		if strings.Index(cmdName,"[") == 0 {
			myutils.Print("Error","the field command-name of " + key + " is invalid,it should a string rather than array",true)
		}
		if cmdName == "" {
			myutils.Print("Error","the field command-name of " + key + " is null",true)
		}
		if value.Get("limit-type").Exists() {
			ikey := value.Get("limit-type").String()
			if _,ok := vd.ResourcesLimits[ikey];ok {
				copy(limit,vd.ResourcesLimits[ikey])
			}
		}
		if value.Get("request-type").Exists() {
			ikey := value.Get("request-type").String()
			if _,ok := vd.ResourcesRequests[ikey];ok {
				copy(request,vd.ResourcesRequests[ikey])
			}
		}
		if value.Get("limit").Exists() {
			for k,t := range value.Get("limit").Array() {
				limit[k] = t.String()
			}		
		}
		if value.Get("request").Exists() {
			for k,t := range value.Get("request").Array() {
				request[k] = t.String()
			}		
		}
		for i := 0 ;i < 2;i++ {
			if i == 0 {
				CheckResourceValue(limit[i],key,"m")	
				CheckResourceValue(request[i],key,"m")	
			}else {
				CheckResourceValue(limit[i],key,"Mi")	
				CheckResourceValue(request[i],key,"Mi")	
			}
		}
		vd.NormData[key] = &Step{Container: container,CommandName: cmdName,ResourcesLimits: limit,ResourcesRequests: request}
		
	}
	
}
func CheckResourceValue(item,step,unit string) {
	if item == "" {
		return
	}
	tdata := strings.Split(item,unit)
	if len(tdata) == 2 && tdata[1] == ""  {
		_,err := strconv.Atoi(tdata[0])
		if err != nil {
			fmt.Printf("invalid value %s of resources in %s,should like %s",item,step,"200" + unit)
			os.Exit(3)
		}else {
			return 
		}
	}
	fmt.Printf("invalid value %s of resources in %s,should like %s",item,step,"200" + unit)
	os.Exit(3)
}
func ArrayToString(key string,value gjson.Result) []string {
	redata := make([]string,len(value.Get(key).Array()))
	for _,name := range value.Get(key).Array() {
		
		redata = append(redata,name.String())
	}
	return redata
}
func (vd *Validator) ValidateResourcesValue() {
	for key,value := range vd.ResourcesLimits {
		CheckResourceValue(value[0],key,"m")
		CheckResourceValue(value[1],key,"Mi")
	}

}
func (vd *Validator) ValidateStepItem() {
	data := vd.Data
	if len(data) == 0 {
		myutils.Print("Error","json object is null,parse failed",true)
	}
	for key,value := range data {
		if !value.Get("command").Exists() {
			myutils.Print("Error", key + " has no field:command,exit",true)
		}
		if !value.Get("container").Exists() {
			myutils.Print("Error",key + " has no field:container,exit",true)
		}
		if !value.Get("pre-steps").Exists() {
			myutils.Print("Error",key + " has no field:pre-steps,exit",true)
		}
		if !value.Get("sub-args").Exists() {
			myutils.Print("Error",key + " has no field:sub-args,exit",true)
		}
		if !value.Get("args").Exists() {
			myutils.Print("Error",key + " has no field:args,exit",true)
		}
		if !value.Get("command-name").Exists() {
			myutils.Print("Error",key + " has no field:command-name,exit",true)
		}
	}
}

func NewValidator(data,referPath,dataPath string,params map[string]string) (*Validator,error) {
	referData := make(map[string]string)
	inputData := make([]string,0,100)
	rawData := make(map[string]gjson.Result)
	normData := make(map[string]*Step)
	limit := make(map[string][]string)
	request := make(map[string][]string)
	if ! gjson.Valid(data) {
		return nil,fmt.Errorf("invalid pipeline file,parse failed,some commas are add in bad area or don't delete the annotation?")
	}
	jsonObj := gjson.Parse(data)
	jsonObj.ForEach(func(key,value gjson.Result)bool{
		name := strings.Trim(key.String()," ")
		if name == "refer" {
			value.ForEach(func(ikey, ivalue gjson.Result) bool {
				referData[ikey.String()] = path.Join(referPath,ivalue.String())
				return true
            })
		}else if name == "params" {
			value.ForEach(func(ikey,ivalue gjson.Result) bool {
 				if _,ok := params[ikey.String()];!ok  {
                    params[ikey.String()] = ivalue.String()
                }   
                return true
            }) 
		}else if name == "input" {
			value.ForEach(func(ikey,ivalue gjson.Result) bool {
				inputData = append(inputData,ivalue.String())
                return true
            })
		}else if strings.Contains(name,"step") {
			var step string
			if name == "step0" {
				myutils.Print("Error","step0 can't be assigned,it is a initialization step",true)
			}else if strings.Index(name,"step") != 0 {
				myutils.Print("Error","step " + name + " is invalid,must be as follows:step1,step2,step3...",true)
			}
			ind,err := GetStepIndex(name)
			if  err != nil {
				myutils.Print("Error","step " + name + " is invalid,must be as follows:step1,step2,step3...",true)
			}else {
				step = "step" + strconv.Itoa(ind)
			}
			if _,ok := rawData[step]; ok {
				myutils.Print("Error","step " + name + " redefined,please make sure that the step is unique",true)
			}
			rawData[step] = value

		}else if strings.Index(name,"resources-limits") == 0 {
			tstr := make([]string,2,2)
			if value.Get("cpu").Exists() {
				tstr[0] = value.Get("cpu").String()
			}
			if value.Get("memory").Exists() {
				tstr[1] = value.Get("memory").String()
			}
			limit[name] = tstr
		}else if strings.Index(name,"resources-requests") == 0 {
			tstr := make([]string,2,2)
			if value.Get("cpu").Exists() {
				tstr[0] = value.Get("cpu").String()
			}
			if value.Get("memory").Exists() {
				tstr[1] = value.Get("memory").String()
			}
			request[name] = tstr
		}
		return true 
	})
	return &Validator{
		NormData: normData,
		Data: rawData,
		ResourcesLimits: limit,
		ResourcesRequests: request,
		Refer: referData,
		Param: params,
		BaseDir: dataPath,
		Input: inputData},nil
}
