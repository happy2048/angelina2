package validator
import(
    "strings"
    "fmt"
	"path"
    "strconv"
    "myutils"
    "regexp"
 	"encoding/json"
    gjson "github.com/tidwall/gjson"
)
type Step struct {
    Command string `json:command`
	TmpArgs []string 
	CommandName string `json:commandName`
    Args string `json:args`
    Container string `json:container`
    Prestep []string `json:prestep`
    SubArgs []string `json:subArgs`
	CmdArray []string 
	CommandCount string `json:commandCount`
}
type Validator struct {
	Data map[string]gjson.Result
	Refer map[string]string
	Param map[string]string
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
	vd.ValidateField()
	vd.SetTmpStepValue()
	vd.ValidatePreStep()
	vd.ReplaceFlag()
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
		if strings.Trim(container," ") == "" {
			myutils.Print("Error","the field container of " + key + " is null",true)
		}
		if strings.Trim(cmdName," ") == "" {
			myutils.Print("Error","the field command-name of " + key + " is null",true)
		}
		vd.NormData[key] = &Step{Container: container,CommandName: cmdName}
		
	}
	
}
func ArrayToString(key string,value gjson.Result) []string {
	redata := make([]string,len(value.Get(key).Array()))
	for _,name := range value.Get(key).Array() {
		
		redata = append(redata,name.String())
	}
	return redata
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

		}
		return true 
	})
	return &Validator{
		NormData: normData,
		Data: rawData,
		Refer: referData,
		Param: params,
		BaseDir: dataPath,
		Input: inputData},nil
}
