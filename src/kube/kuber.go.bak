package kube
import(
    "myutils"
	"redisdb"
	"fmt"
	"io/ioutil"
    apiv1 "k8s.io/api/core/v1"
    exv1beta "k8s.io/api/extensions/v1beta1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)
/*
* CreateDeployArgs: 创建deployment传递参数使用
*/
type PodPhase string

const (
    // PodPending means the pod has been accepted by the system, but one or more of the containers
    // has not been started. This includes time before being bound to a node, as well as time spent
    // pulling images onto the host.
    PodPending PodPhase = "Pending"
    // PodRunning means the pod has been bound to a node and all of the containers have been started.
    // At least one container is still running or is in the process of being restarted.
    PodRunning PodPhase = "Running"
    // PodSucceeded means that all containers in the pod have voluntarily terminated
    // with a container exit code of 0, and the system is not going to restart any of these containers.
    PodSucceeded PodPhase = "Succeeded"
    // PodFailed means that all containers in the pod have terminated, and at least one container has
    // terminated in a failure (exited with a non-zero exit code or was stopped by the system).
    PodFailed PodPhase = "Failed"
    // PodUnknown means that for some reason the state of the pod could not be obtained, typically due
    // to an error in communicating with the host of the pod.
    PodUnknown PodPhase = "Unknown"
)
type DeploymentStatus int
const (
	_ DeploymentStatus = iota
	NotFound
	Available
	UnAvailable
)
type CreateDeployArgs struct {
	Sample string  // 属于哪一个sample
	Step  string  // 初始化容器时使用,
	Index string
	DeployId string
	Container string //创建deployment使用到的container
}
type K8sClient struct {
	ClientSet *kubernetes.Clientset
	Args *Config
}
type Config struct {
	AuthFile string //认证文件路径
	ReferVolume string // glusterfs refer volume名称
	DataVolume string  // glusterfs data volume 名称
	EndpointsName string // glusterfs endpoints 名称
	RedisAddr string    //redis address
	ControllerService   string 
	ScriptUrl string   // 各个容器需要下载的script地址 
	NameSpace string // kubernetes 名称空间
	DataDir string  // data dir 在容器中路径，默认为/mnt/data
	ReferDir string // refer dir 在容器中路径 ，默认为/mnt/refer
	StartRunCmd string  // 容器开始运行的命令，默认为/bin/bash /usr/bin/rundoc.sh
}
func  NewK8sClient(redisAddr string) *K8sClient {
	db := redisdb.NewRedisDB("tcp",redisAddr)
    authString,_ := db.RedisHashGet("kubernetesConfig","AuthFile")
    referVolume,_ := db.RedisHashGet("kubernetesConfig","ReferVolume")
    dataVolume,_ := db.RedisHashGet("kubernetesConfig","DataVolume")
    endpoints,_ := db.RedisHashGet("kubernetesConfig","GlusterEndpoints")
    namespace,_ := db.RedisHashGet("kubernetesConfig","Namespace")
    scriptUrl,_ := db.RedisHashGet("kubernetesConfig","ScriptUrl")
    startRunCmd,_ := db.RedisHashGet("kubernetesConfig","StartRunCmd")
    ctrlService,_ := db.RedisHashGet("kubernetesConfig","ControllerServiceEntry")
    dataDir := "/mnt/data"
    referDir := "/mnt/refer"
    auth := "/root/.kube.tmp.conf"
	if scriptUrl == "" {
		scriptUrl = "http://" + ctrlService + "/angelina-runner"
	}
    ioutil.WriteFile(auth,[]byte(authString),0644)
	st := initClientSets(auth)
	td := &Config{
		AuthFile: auth,
		ReferVolume: referVolume,
		DataVolume: dataVolume,
		EndpointsName: endpoints,
		RedisAddr: redisAddr,
		ScriptUrl: scriptUrl,
		NameSpace: namespace,
		DataDir: dataDir,
		ControllerService: ctrlService,
		StartRunCmd: startRunCmd,
		ReferDir: referDir}
	return &K8sClient {
		ClientSet: st,
		Args: td}
}
/*
func (k8s *K8sClient) CreateController(sample string) (bool,error) {
	deploymentId := myutils.GetSamplePrefix(sample)
	deployment := &exv1beta.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentId,
			Namespace: k8s.Args.NameSpace,
		},
		Spec: exv1beta.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentId,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentId,
					},
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: "refer-volume",
							VolumeSource: apiv1.VolumeSource{
								Glusterfs: &apiv1.GlusterfsVolumeSource {
									EndpointsName: k8s.Args.EndpointsName,
									Path: k8s.Args.ReferVolume,
									ReadOnly: true,
									
								},
							},
						},	
						{
							Name: "data-volume",
							VolumeSource: apiv1.VolumeSource{
								Glusterfs: &apiv1.GlusterfsVolumeSource {
									EndpointsName: k8s.Args.EndpointsName,
									Path: k8s.Args.DataVolume,
									ReadOnly: false,
									
								},
							},
						},	
					},
					Containers: []apiv1.Container{
						{
							Name:  deploymentId,
							Image: k8s.Args.ControllerContainer,
							Env: []apiv1.EnvVar{
								{
									Name: "REDISADDR",
									Value: k8s.Args.RedisAddr,
								},
								{
									Name: "SAMPLE",
									Value: sample,
								},
								{
									Name: "DEPLOYMENTID",
									Value: deploymentId,
								},				
								{
									Name: "DATADIR",
									Value: k8s.Args.DataDir,
								},
								{
									Name: "REFERDIR",
									Value: k8s.Args.ReferDir,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:  "refer-volume",
									MountPath: k8s.Args.ReferDir,
								},
								{
									Name:  "data-volume",
									MountPath: k8s.Args.DataDir,
								},
							},
						},
					},
				},
			},
		},
	}
	_,err := k8s.ClientSet.ExtensionsV1beta1().Deployments(k8s.Args.NameSpace).Create(deployment)
	if err != nil {
		myutils.Print("Error"," Create Deployment Failed,reason:" + err.Error(),false)
		return false,err
	}else {
		return true,err
	}		

}
*/
func (k8s *K8sClient) CreateDeployment(cda *CreateDeployArgs) (bool,error)  {
	deploymentId := cda.DeployId
	prefix := myutils.GetSamplePrefix(cda.Sample)
	deployment := &exv1beta.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentId,
			Namespace: k8s.Args.NameSpace,
		},
		Spec: exv1beta.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentId,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentId,
					},
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: "refer-volume",
							VolumeSource: apiv1.VolumeSource{
								Glusterfs: &apiv1.GlusterfsVolumeSource {
									EndpointsName: k8s.Args.EndpointsName,
									Path: k8s.Args.ReferVolume,
									ReadOnly: true,
									
								},
							},
						},	
						{
							Name: "data-volume",
							VolumeSource: apiv1.VolumeSource{
								Glusterfs: &apiv1.GlusterfsVolumeSource {
									EndpointsName: k8s.Args.EndpointsName,
									Path: k8s.Args.DataVolume,
									ReadOnly: false,
									
								},
							},
						},	
					},
					Containers: []apiv1.Container{
						{
							Name:  deploymentId,
							Image: cda.Container,
							Command: []string{k8s.Args.StartRunCmd},
							Env: []apiv1.EnvVar{
								{
									Name: "REDISADDR",
									Value: k8s.Args.RedisAddr,
								},
								{
									Name: "SAMPLE",
									Value: cda.Sample,
								},
								{
									Name: "DEPLOYMENTID",
									Value: deploymentId,
								},				
								{
									Name: "SERVICE",
									Value: k8s.Args.ControllerService,
								},				
								{
									Name: "SCRIPTURL",
									Value: k8s.Args.ScriptUrl,
								},
								{
									Name: "STEP",
									Value: cda.Step,
						
								},
								{
									Name: "INDEX",
									Value: cda.Index,
						
								},
								{
									Name: "SENDMESSAGECHAN",
									Value:  prefix + "__" + "ReceiveFromCon",
								},
								{
									Name: "RECEIVEMESSAGECHAN",
									Value: prefix + "__" + "SendToCon",
								},
								{
									Name: "DATADIR",
									Value: k8s.Args.DataDir,
								},
								{
									Name: "REFERDIR",
									Value: k8s.Args.ReferDir,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:  "refer-volume",
									MountPath: k8s.Args.ReferDir,
								},
								{
									Name:  "data-volume",
									MountPath: k8s.Args.DataDir,
								},
							},
						},
					},
				},
			},
		},
	}
	_,err := k8s.ClientSet.ExtensionsV1beta1().Deployments(k8s.Args.NameSpace).Create(deployment)
	if err != nil {
		myutils.Print("Error"," Create Deployment Failed,reason:" + err.Error(),false)
		return false,err
	}else {
		return true,err
	}
}
func (k8s *K8sClient) DeleteDeployment(deploymentId string) (bool,error) {
	deletePolicy := metav1.DeletePropagationForeground
    err := k8s.ClientSet.ExtensionsV1beta1().Deployments(k8s.Args.NameSpace).Delete(deploymentId, &metav1.DeleteOptions{
        PropagationPolicy: &deletePolicy,
    }) 
	if  err != nil {
        myutils.Print("Error"," Delete deployment failed",false)
		return false,err
    }
    myutils.Print("Info"," Delete deployment succeed",false)
	return true,err
}
func (k8s *K8sClient) DeploymentExist(deploymentId string) (DeploymentStatus) {
	dep, err := k8s.ClientSet.ExtensionsV1beta1().Deployments(k8s.Args.NameSpace).Get(deploymentId, metav1.GetOptions{})
	if err != nil {
		return NotFound
	}
	if dep.Status.AvailableReplicas > 0 {
		return Available
	}
	return UnAvailable
	
}
func (k8s *K8sClient) PodsExist(key string) (bool,error) {
	pods, err := k8s.ClientSet.CoreV1().Pods(k8s.Args.NameSpace).List(metav1.ListOptions{})
	if err != nil {
		return false,err
	}
	if len(pods.Items) == 0 {
		return false,nil
	}

	for _,pod := range pods.Items {
		fmt.Println(pod.Status.Phase)
		for _,val := range pod.Spec.Containers {
			if val.Name == key {
				return true,nil
			}
		}
	}
	return false,nil
}
func (k8s *K8sClient) GetPod(key string) {
	data,err := k8s.ClientSet.CoreV1().Pods(k8s.Args.NameSpace).Get(key,metav1.GetOptions{})
	fmt.Println(data)
	fmt.Println(data.Spec)
	fmt.Println(data.Status.Reason)
	fmt.Println(err)

}
func (k8s *K8sClient) WatchPod()  {
	data,err := k8s.ClientSet.CoreV1().Pods(k8s.Args.NameSpace).Watch(metav1.ListOptions{})
	fmt.Println(data)
	fmt.Println(err)
}
func  initClientSets(k8sconfig string) (*kubernetes.Clientset)  {
    config, err := clientcmd.BuildConfigFromFlags("", k8sconfig)
    if err != nil { 
		myutils.Print("Error","read kubernetes configure file failed",true)
    }
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
		myutils.Print("Error","create clientset failed",false)
    }
    return clientset
}
func int32Ptr(i int32) *int32 { return &i }
