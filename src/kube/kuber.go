package kube
import(
	"net/http"
	"strings"
	"fmt"
	"strconv"
	"path"
	"encoding/json"
	"crypto/tls"
	"io/ioutil"
	gjson "github.com/tidwall/gjson"
)
type ReadQuota struct {
	LimitsCpu string `json:"limits.cpu"`
	LimitsMemory string `json:"limits.memory"`
	RequestsCpu string `json:"requests.cpu"`
	RequestsMemory string `json:"requests.memory"`
	LimitPods string `json:"pods"`
}
type DeploymentStatus int
const (
    _ DeploymentStatus = iota
    NotFound
    Running
	Pending
    UnAvailable
	OtherError
)
type CreateDeployArgs struct {
	Sample string  // 属于哪一个sample
	Step  string  // 初始化容器时使用,
	Index string
	DeployId string
	Container string //创建deployment使用到的container
	Limits []string
	Requests []string
}
type InitArgs struct {
	ApiServer string
	ControllerEntry string
	DeploymentTemp  string
	Namespace string
	RedisAddr string
	StartCmd string
	QuotaName string
	GlusterfsEndpoint string
	GlusterfsDataVolume string
	GlusterfsReferVolume string
}
type Kube struct {
	ApiServer string
	Protocol  string
	RedisAddr string
	GlusterfsEndpoint string
	GlusterfsDataVolume string
	GlusterfsReferVolume string
    AngelinaControllerEntry string
	DeploymentTemplate string
	Namespace string
	StartCmd string
	QuotaName string

}
func NewKube(init *InitArgs) *Kube {
	protocol := strings.Split(init.ApiServer,"://")[0]
	return &Kube{
		ApiServer: init.ApiServer,
		Protocol: protocol,
		StartCmd: init.StartCmd,
		Namespace: init.Namespace,
		QuotaName: init.QuotaName,
		RedisAddr: init.RedisAddr,
		GlusterfsEndpoint: init.GlusterfsEndpoint,
		GlusterfsDataVolume: init.GlusterfsDataVolume,
		GlusterfsReferVolume: init.GlusterfsReferVolume,
		AngelinaControllerEntry: init.ControllerEntry,
		DeploymentTemplate: init.DeploymentTemp}
}
func (k8s *Kube) CreateDeployment(cda *CreateDeployArgs) error {
	tdata := k8s.DeploymentTemplate
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-NAME",cda.DeployId,-1)
	tdata = strings.Replace(tdata,"ANGELINA-NAMESPACE",k8s.Namespace,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-IMAGE",cda.Container,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-COMMAND",k8s.StartCmd,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REDIS",k8s.RedisAddr,-1)
	if cda.Requests[0] == "" {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REQUESTS-CPU","0m",-1)
	}else {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REQUESTS-CPU",cda.Requests[0],-1)
	}
	if cda.Requests[1] == "" {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REQUESTS-MEMORY","0Mi",-1)
	}else {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REQUESTS-MEMORY",cda.Requests[1],-1)
	}
	delete := 0
	if cda.Limits[0] == "" {
		if cda.Requests[0] == "" {
			tdata = strings.Replace(tdata,"ANGELINA-RUNNER-LIMITS-CPU","0m",-1)
		}else {
			tdata = strings.Replace(tdata,"cpu: ANGELINA-RUNNER-LIMITS-CPU","",-1)
			delete++
		}
	}else  {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-LIMITS-CPU",cda.Limits[0],-1)
	}
	if cda.Limits[1] == "" {
		if cda.Requests[1] == "" {
			tdata = strings.Replace(tdata,"ANGELINA-RUNNER-LIMITS-MEMORY","0Mi",-1)
		}else {
			tdata = strings.Replace(tdata,"memory: ANGELINA-RUNNER-LIMITS-MEMORY","",-1)
			delete++
		}
	}else {
		tdata = strings.Replace(tdata,"ANGELINA-RUNNER-LIMITS-MEMORY",cda.Limits[1],-1)
	}
	
	if delete == 2 {
		tdata = strings.Replace(tdata,"limits:","",-1)
	}
	/*
	if cda.Requests[0] == cda.Requests[1] && cda.Requests[1] == "" {
		tdata = strings.Replace(tdata,"requests:","",-1)
	}
	if strings.Index(tdata,"limits:") == -1 && strings.Index(tdata,"requests:") == -1 {
		tdata = strings.Replace(tdata,"resources:","",-1)
	}
	*/
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-JOB",cda.Sample,-1)
	tdata = strings.Replace(tdata,"ANGELINA-CONTROLLER-ENTRY",k8s.AngelinaControllerEntry,-1)
	scriptUrl := "http://" + k8s.AngelinaControllerEntry + "/angelina-runner"
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-SCRIPTURL",scriptUrl,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-STEP",cda.Step,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-INDEX",cda.Index,-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-DATADIR","/mnt/data",-1)
	tdata = strings.Replace(tdata,"ANGELINA-RUNNER-REFERDIR","/mnt/refer",-1)
	tdata = strings.Replace(tdata,"ANGELINA-GLUSTERFS-ENDPOINT",k8s.GlusterfsEndpoint,-1)
	tdata = strings.Replace(tdata,"ANGELINA-GLUSTERFS-DATA-VOLUME",k8s.GlusterfsDataVolume,-1)
	tdata = strings.Replace(tdata,"ANGELINA-GLUSTERFS-REFER-VOLUME",k8s.GlusterfsReferVolume,-1)
	return k8s.PostInfo(tdata)
}
func (k8s *Kube) DeleteDeployment(deploymentId string) error {
	//subUrl := path.Join("apis/apps/v1beta1/namespaces",k8s.Namespace,"deployments",deploymentId)
	subUrl := path.Join("api/v1/namespaces",k8s.Namespace,"pods",deploymentId)
	url := strings.Trim(k8s.ApiServer,"/") + "/" + subUrl 
	response,err := k8s.HttpOperate("DELETE",url,"",nil)
    if err != nil {
        return err
    }
	if response.StatusCode >= 200 && response.StatusCode <= 300 {
		return nil
	}
	if response.StatusCode == 404 {
		return fmt.Errorf("%s","deployment not found")
	}
	body,err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error: read response body error,reason:%s",err.Error())
	}
    return fmt.Errorf("%s",string(body))
}
func TranMemUnit(item string) int64 {
	item = strings.Trim(item," ")
	if item == "" {
		return 0
	}
	if strings.HasSuffix(item,"Ki") {
		value := ParseValue(item,"Ki")
		if value == int64(-1) {
			return value
		}
		return value * int64(1 << 10)
	}
	if strings.HasSuffix(item,"Mi") {
		value := ParseValue(item,"Mi")
		if value == int64(-1) {
			return value
		}
		return value * int64(1 << 20)
	}
	if strings.HasSuffix(item,"Gi") {
		value := ParseValue(item,"Gi")
		if value == int64(-1) {
			return value
		}
		return value * (1 << 30)
	}
	if strings.HasSuffix(item,"Ti") {
		value := ParseValue(item,"Ti")
		if value == int64(-1) {
			return value
		}
		return value * int64(1  << 40)
	}
	data,err := strconv.ParseInt(item,10,64)
	if err != nil {
		return int64(-1)
	}
	return data
}
func TranPods(item string) int64 {
	item = strings.Trim(item," ")
	if item == "" {
		return 0
	}
	data,err := strconv.ParseInt(item,10,64)
	if err != nil {
		return int64(-1)
	}
	return data
}
func TranCpuUnit(item string) int64 {
	item = strings.Trim(item," ")
	if item == "" {
		return 0
	}
	if strings.HasSuffix(item,"m") {
		value := ParseValue(item,"m")
		if value == int64(-1) {
			return value
		}
		return value
	}
	data,err := strconv.ParseInt(item,10,64)
	if err != nil {
		return int64(-1)
	}
	return data * 1000
}
func ParseValue(item,unit string) int64 {
	t := strings.Split(item,unit)
	if len(t) == 2 && t[1] == "" {
		value,err := strconv.ParseInt(t[0],10,64)
		if err != nil {
			return int64(-1)
		}
		return value
	}
	return int64(-1)
}
func (k8s *Kube) GetQuotaResources(myCpus,myMems string) (error) {
	subUrl := path.Join("api/v1/namespaces",k8s.Namespace,"resourcequotas",k8s.QuotaName,"status")
	url :=  strings.Trim(k8s.ApiServer,"/") + "/" + subUrl
	totalCpus := int64(0)
	totalMem := int64(0)
	totalPods := int64(0)
	reInfo := make(map[string]ReadQuota)
	response,err := k8s.HttpOperate("GET",url,"",nil)
	if err != nil {
		return err
	}
	body,err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Error: read response body error,reason:%s",err.Error())
	}
	if response.StatusCode == 404 {
		return nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
    	return fmt.Errorf("%s",string(body))
	}
	jsonObj := gjson.Parse(string(body))
	jsonObj.ForEach(func(key,value gjson.Result) bool {
		if key.String() == "status" {
			if value.Get("hard").Exists() {
				var mydata ReadQuota
				json.Unmarshal([]byte(value.Get("hard").String()),&mydata)
				reInfo["hard"] = mydata
			}				
			if value.Get("used").Exists() {
				var mydata ReadQuota
				json.Unmarshal([]byte(value.Get("used").String()),&mydata)
				reInfo["used"] = mydata
			}				
		}
		return true 
	})
	_,ok1 := reInfo["hard"]
	_,ok2 := reInfo["used"]
	if ok1 && ok2 {
		totalCpus = TranCpuUnit(reInfo["hard"].RequestsCpu) - TranCpuUnit(reInfo["used"].RequestsCpu)
		totalMem = TranMemUnit(reInfo["hard"].RequestsMemory) - TranMemUnit(reInfo["used"].RequestsMemory)
		totalPods = TranPods(reInfo["hard"].LimitPods)  - TranPods(reInfo["used"].LimitPods)
	}else {
		return nil
	}
	if reInfo["hard"].LimitPods != "" && totalPods <= 0 {
		return fmt.Errorf("no resource to allocate")
	}
	if TranMemUnit(reInfo["hard"].RequestsMemory) == 0 && TranCpuUnit(reInfo["hard"].RequestsCpu) == 0 {
		return nil
	}
	if TranCpuUnit(reInfo["hard"].RequestsCpu) == 0 && totalMem >= TranMemUnit(myMems) {
		return nil
	}
	if TranMemUnit(reInfo["hard"].RequestsMemory) == 0 && totalCpus >= TranCpuUnit(myCpus) {
		return nil
	}
	if totalCpus >= TranCpuUnit(myCpus) && totalMem >= TranMemUnit(myMems) {
		return nil
	}
	return fmt.Errorf("no resource to allocate")

}
func (k8s *Kube) DeploymentExist(deploymentId string) DeploymentStatus {
	//subUrl := path.Join("apis/apps/v1beta1/namespaces",k8s.Namespace,"deployments",deploymentId,"status?pretty=true")
	subUrl := path.Join("api/v1/namespaces",k8s.Namespace,"pods",deploymentId,"status")
	url := strings.Trim(k8s.ApiServer,"/") + "/" + subUrl
	response,err := k8s.HttpOperate("GET",url,"",nil)
	reStatus := NotFound
	if err != nil {
		return OtherError
	}
	if response.StatusCode == 404 {
		return reStatus
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return OtherError
	}
	jsonObj := gjson.Parse(string(body))
	jsonObj.ForEach(func(key,value gjson.Result) bool {
		if key.String() == "status" {
			status := value.Get("phase").String()
			if status == "Running" {
				reStatus = Running
			}else if status == "Pending" {
				reStatus = Pending
			}else {
				reStatus = UnAvailable
			}
		}
		return true
	})
	return reStatus
}
func (k8s *Kube) HttpOperate(method,url,data string,header map[string]string) (*http.Response,error) {
	var client *http.Client
	if k8s.Protocol == "https" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify : true},
		}
		client = &http.Client{Transport: tr}
		
	}else {
		client = &http.Client{}
	} 
	var request *http.Request
	var err error
	if data == "" {
		request,err = http.NewRequest(method,url,nil)
	}else {
		request,err = http.NewRequest(method,url,strings.NewReader(data))
	}
	if err != nil {
		return nil,err
	}
	for key,value := range header {
    	request.Header.Set(key,value)
	}
	response,err := client.Do(request)
	return response,err
}
/*
func main() {
   for i := 1 ;i < 5;i++ {
	Post("yang"+ strconv.Itoa(i),"create")
   }
	//GetInfo()
	//PostInfo()
	bdata,_ := ioutil.ReadFile("/root/biofile/test/angelina-runner-deployment.yml")
	data := string(bdata)
	init := &InitArgs{
		ApiServer: "https://10.61.0.86:6443",
		ControllerEntry: "angelina-controller:6300",
		Namespace: "bio-system",
		StartCmd: "rundoc.sh",
		DeploymentTemp: data,
		GlusterfsEndpoint: "glusterfs-cluster",
		GlusterfsDataVolume: "data-volume",
		GlusterfsReferVolume: "refer-volume"}
	k8s := NewKube(init) 
	create := &CreateDeployArgs {
		Sample: "test1",
		Step: "step2",
		Index: "1",
		Container: "happy365/angelina-controller:2.0",
		DeployId: "mytest1",
		Limits: []string{"34",""},
		Requests: []string{"","45"}}
	fmt.Println(create)
	sdata := k8s.CreateDeployment(create)
	fmt.Println(sdata)
	//fmt.Println(k8s.DeleteDeployment("mytest1"))
	re := k8s.DeploymentExist("mytest1")
	fmt.Println(re)
	avc,err := k8s.GetNodesResources(50000,222222222)
	fmt.Println(avc,err)
}
*/
func (k8s *Kube) PostInfo(info string) error {
	//subUrl := path.Join("apis/apps/v1beta1/namespaces",k8s.Namespace,"deployments")
	subUrl := path.Join("api/v1/namespaces",k8s.Namespace,"pods")
	url := strings.Trim(k8s.ApiServer,"/") + "/" + subUrl
	header := make(map[string]string)
	header["Content-Type"] = "application/yaml" 
	response,err := k8s.HttpOperate("POST",url,info,header)
    if err != nil {
        return err
    }
	if response.StatusCode >= 200 && response.StatusCode <= 300 {
		return nil
	}
	if response.StatusCode == 409 {
		return fmt.Errorf("Error: deployment already exists")
	}
	body,_ := ioutil.ReadAll(response.Body)
	return fmt.Errorf("%s",string(body))
}
