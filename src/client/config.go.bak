package client
import (
	"strings"
	"os"
	"myutils"
	"io/ioutil"
	gjson "github.com/tidwall/gjson"
)
func (cc *Connector) ReadConfig(data string) {
	config := make(map[string]string)
	jsonObj := gjson.Parse(data)
	jsonObj.ForEach(func(key,value gjson.Result) bool {
		if key.String() == "AuthFile" {
		 	info,err := os.Stat(value.String())
			if err == nil {
				if info.IsDir() {
					myutils.Print("Error","read kubernetes auth file failed,it's a directory",true)
				}
				auth,err1 := ioutil.ReadFile(value.String())
				if err1 != nil {
					myutils.Print("Error","read kubernetes auth file failed,exit",true)					
				} 
				config["AuthFile"] = string(auth)
			}else if os.IsNotExist(err){
				myutils.Print("Error","read kubernetes auth file failed,it's not exist",true)
			}else {
				myutils.Print("Error","read kubernetes auth file failed",true)
				
			}
		}else if key.String() == "ReferVolume" {
			config["ReferVolume"] = value.String()
		}else if key.String() == "DataVolume" {
			config["DataVolume"] = value.String()
		}else if key.String() == "GlusterEndpoints" {
			config["GlusterEndpoints"] = value.String()
		}else if key.String() == "Namespace" {
			config["Namespace"] = value.String()
		}else if key.String() == "ScriptUrl" {
			config["ScriptUrl"] = value.String()
		}else if key.String() == "OutputBaseDir" {
			config["OutputBaseDir"] = value.String()
		}else if key.String() == "StartRunCmd" {
			config["StartRunCmd"] = value.String()
		}else if key.String() == "ControllerServiceEntry" {
			config["ControllerServiceEntry"] = value.String()
		}else {
			myutils.Print("Warning","config file no this key: " + key.String(),false)
		
		}
		return true	
	})
	keys := []string{"DataVolume","ReferVolume","ScriptUrl","StartRunCmd","AuthFile","GlusterEndpoints","Namespace","OutputBaseDir","ControllerServiceEntry"}
	for _,val := range keys {
		if _,ok := config[val]; !ok {
			status,_ := cc.Db.RedisHashGet("kubernetesConfig",val)
			if status == "" && val != "ScriptUrl" {
				myutils.Print("Error","config file not give " + val,true)
			}
		}else {
			if strings.Trim(config[val]," ") == "" {
				status,_ := cc.Db.RedisHashGet("kubernetesConfig",val)
            	if status == "" && val != "ScriptUrl" {
                	myutils.Print("Error","config file not give " + val,true)
            	}else {
					delete(config,val)
				}
			}	 
		} 
	}
	for key,val := range config {
		_,err := cc.Db.RedisHashSet("kubernetesConfig",key,val)
		if err != nil {
			myutils.Print("Error","set " + key + " to redis failed",true)
		}
	}
} 
