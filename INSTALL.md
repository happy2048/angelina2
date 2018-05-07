安装：

**1.kubernetes安装**

kubernetes安装可以参考：[kuberntes安装](https://github.com/happy2048/k8s-config-files)

**2.glusterfs安装**

下面说明整个集群的部署信息（请根据自身实际情况做修改）：

| ip  | role  | hostname  |
| ------------ | ------------ | ------------ |
| 10.61.0.91  | glusterfs server  |  vmnode1  |
| 10.61.0.92 | glusterfs server  |  vmnode2 |
| 10.61.0.93 | glusterfs server  |  vmnode3|
| 10.61.0.160  | glusterfs client, kubernetes master  | kuber-master  |
| 10.61.0.161  |  glusterfs client,kubernetes node |  kuber-node1 |
| 10.61.0.162 |  glusterfs client,kubernetes node |  kuber-node2 |
| 10.61.0.163 |  glusterfs client,kubernetes node |  kuber-node3 |

（1） 在 vmnode1,vmnode2,vmnode3上执行如下步骤：

	yum install -y glusterfs glusterfs-server glusterfs-fuse glusterfs-rdma

（2） 在vmnode1,vmnode2,vmnode3上启动glusterFS:

	systemctl start glusterd
	systemctl enable glusterd

（3） 在vmnode1上执行如下步骤，把vmnode2,vmnode3加入到集群当中：

	gluster peer probe vmnode1
	gluster peer probe vmnode2

（4） 在gluster1查看集群状态:

	[root@vmnode1 ~]# gluster peer status
	Number of Peers: 2

	Hostname: vmnode2
	Uuid: 3a6c9d1a-eb85-49e6-8a71-faf86b78d653
	State: Peer in Cluster (Connected)

	Hostname: vmnode3
	Uuid: 730ab35b-e2a3-4f7f-a718-5d9e1d7f49d9
	State: Peer in Cluster (Connected)

（5） 在vmnode1,vmnode2,vmnode3执行如下命令，创建数据存储目录：

	mkdir  /mnt/gluster-data
	mkdir  /mnt/gluster-refer

（6） 创建volume,这里创建两个volume,一个是data-volume,一个是refer-volume,data-volume是用来存放分析数据的，refer-volume是存放参考数据的，比如参考基因组文件，下面命令只需在vmnode1上执行就行：

	gluster volume create data-volume replica 3 vmnode1:/mnt/gluster-data  vmnode2:/mnt/gluster-data vmnode3:/mnt/gluster-data

	gluster volume create refer-volume replica 3 vmnode1:/mnt/gluster-refer  vmnode2:/mnt/gluster-refer vmnode3:/mnt/gluster-refer
	gluster volume create redis-volume replica 3 vmnode1:/mnt/gluster-redis  vmnode2:/mnt/gluster-redis vmnode3:/mnt/gluster-redis
	
（7） 在vmnode1上启动data-volume,refer-volume：

	gluster volume start data-volume
	gluster volume start refer-volume
	gluster voluem start redis-volume
	
（8）在kuber-master,kuber-node1,kuber-node2,kuber-node3安装glusterfs客户端软件，便于容器挂载：

	yum install -y glusterfs-fuse glusterfs

**3.在kubernetes创建gluster service和gluster endpoint**

（1） 为了不影响kubernetes中其他应用，我们另外创建一个namespace,在kuber-master上创建一个文件：
	
	[root@kuber-master ~]# mkdir /opt/angelina
	[root@kuber-master ~]# cat /opt/angelina/bio-system.yml
	apiVersion: v1
	kind: Namespace
	metadata:
       name: bio-system
       labels:
          name: bio-system
（2） 在kuber-master上执行如下命令,创建namespace:

	[root@kuber-master ~]# kubectl apply -f bio-system.yml
	
（3） 在kuber-master上创建gluster service:

	[root@kuber-master angelina]# cat glusterfs-service.json 
	{
	  "kind": "Service",
	  "apiVersion": "v1",
	  "metadata": {
		"name": "glusterfs-cluster",
		"namespace": "bio-system"
	  },
	  "spec": {
		"ports": [
		  {"port": 1}
		]
	  }
	}
	
	[root@kuber-master angelina]# kubectl apply -f glusterfs-service.json

（4） 在kubernetes上创建gluster endpoints,需要说明的是下面的ip是写gluster server的ip，有多少写多少:

	[root@kuber-master angelina]# cat glusterfs-endpoints.json
	{
	  "kind": "Endpoints",
	  "apiVersion": "v1",
	  "metadata": {
		"name": "glusterfs-cluster",
		"namespace": "bio-system"
	  },
	  "subsets": [
		{
		  "addresses": [
			{
			  "ip": "10.61.0.91"
			},
			{
			  "ip": "10.61.0.92"
			},
			{
			  "ip": "10.61.0.93"
			},
		  ],
		  "ports": [
			{
			  "port": 1
			}
		  ]
		}
	  ]
	}
	[root@kuber-master angelina]# kubectl apply -f glusterfs-endpoints.json

**4.在kubernetes上创建redis server**
（1）在kubernetes上创建redis deployment:

	[root@kuber-master angelina]# cat redis-deployment.yml
	apiVersion: apps/v1beta1
	kind: Deployment
	metadata:
	  name: bio-redis
	  namespace: bio-system
	  labels:
		app: bio-redis
	spec:
	  replicas: 2
	  selector:
		matchLabels:
		   app: bio-redis
	  template:
		metadata:
		  labels:
			app: bio-redis
		spec:
		  containers:
		  - name: bio-redis
			image: redis:3.0
			command:
			  - redis-server
			  - "--appendonly"
			  - "yes"
			ports:
			- containerPort: 6379
			volumeMounts:
			- name: data
			  mountPath: /data 
		  volumes:
		  - name: data
			glusterfs:
			  endpoints: glusterfs-cluster
			  path: redis-volume
			  readOnly: false   
	
	[root@kuber-master angelina]# kubectl apply -f redis-deployment.yml
	
（2） 在kubernetes上创建redis service,需要把ip改成相应的ip:

	[root@kuber-master angelina]# cat redis-service.yml 
	apiVersion: v1
	kind: Service
	metadata:
	  name: bio-redis
	  namespace: bio-system
	spec:
	  ports:
	  - port: 6380
		targetPort: 6379
	  selector:
		app: bio-redis
	  type: LoadBalancer
	  externalIPs: 
	  - 10.61.0.86

	[root@kuber-master angelina]# kubectl apply -f redis-service.yml

**5.编译**

在编译之前，请确认golang是否安装。

（1）从github下载angelina：

	[root@kuber-master opt]# git clone https://github.com/happy2048/angelina.git

（2）执行如下命令编译：

	[root@kuber-master opt]# cd  angelina
	[root@kuber-master angelina]#  old=$(echo $GOPATH) && export GOPATH=$(pwd) && make && export GOPATH=$old 

（3）编译完成之后，会在当前目录下生成一个bin目录，如下:

	[root@kuber-master bin]# ll
	total 67240
	-rwxr-xr-x 1 root root 31925505 May  3 20:38 angelina
	-rwxr-xr-x 1 root root 31618999 May  3 20:38 angelina-controller
	-rwxr-xr-x 1 root root  5301877 May  3 20:38 angelina-runner
	
 这三个文件就是angelina的组件。
 
**6.制作angelina controller容器**

（1） 将编译好的angelina-controller单独放在一个文件夹下，并且创建一个Dockerfile，内容如下：

	[root@kuber-master con]# cat Dockerfile 
	FROM centos:7.3.1611
	RUN yum install epel-release -y
	RUN yum install wget -y
	ADD angelina-controller /usr/bin/angelina-controller
	RUN chmod +x /usr/bin/angelina-controller 
	ENTRYPOINT ["angelina-controller"]

（2） 在当前目录下执行如下命令：

	[root@kuber-master con]# docker build -t  angelina-controller:2.0  .    

（3） 将创建好的容器上传到自己本地的容器私有仓库

**7.制作angelina runner容器**

（1） angelina runner是运行任务的容器，每个容器内置的任务命令是不相同的，这里我做一个bwa容器来示范：

	[root@kuber-master bwa]# cat Dockerfile 
	FROM centos:7.3.1611
	RUN yum install epel-release -y
	RUN yum install wget \
		git \
		make \
		gcc \
		gcc-c++ \
		zlib \
		zlib-devel -y
	RUN cd /root && \
		git clone https://github.com/lh3/bwa.git && \
		cd bwa && \
		make && \
		cp bwa /usr/local/bin && \
		cd /root && \
		git clone https://github.com/lh3/minimap2 && \
		cd minimap2 && \
		make && \
		cp minimap2 /usr/local/bin 
	ENV URL http://files.happy366.cn/files/docker/
	ADD angelina-runner  /usr/bin/angelina-runner
	RUN chmod +x /usr/bin/angelina-runner

（2）从上面的Dockerfile中可以看到，每一个任务容器都需要加入angelina-runner这个工具。

（3）将制作好的容器上传到本地私有仓库。

**8.初始化angelina**

（1）初始化之前需要设置REDISADDR环境变量

	[root@kuber-master  angelina]# echo "export REDISADDR=10.61.0.160:6380" >> /root/.bashrc
	[root@kuber-master  angelina]# source  /root/.bashrc
	
（2）使用angelina  -g init 产生模板文件

	[root@kuber-master angelina]# ./bin/angelina -g init 
	create init template file to /tmp/angelina.json

（3）编辑产生的模板文件
	
	[root@kuber-master angelina]#  cat /tmp/angelina.json
	
	{
		"AuthFile": "/etc/kubernetes/admin.conf",  // kubernetes 认证文件路径，创建deployment时需要使用
		"ReferVolume": "refer-volume",  // referVolume是我们刚才创建的refer-volume，存放参考文件的volume
		"DataVolume" : "data-volume", // job使用的gluster volume，也就是我们刚才创建的data-volume
		"GlusterEndpoints": "glusterfs-cluster",// 我们创建glusterfs endpoint时使用的名称
		"Namespace": "bio-system", // 使用我们刚才创建的kubernetes namespace作为angelina专用namespace
		"ScriptUrl": "http://127.0.0.1/angelina-runner", //设置该选项的意义在于，如果我们做了很多关于angelina-runner的容器，如果现在我需要更新angelina-runner，那么这些容器需要从新制作，设置这个url，所有的容器的angelina-runner在运行时从这下载，然后运行。
		"OutputBaseDir": "", // 在该物理机上glusterfs的data-volume挂载点，运行job所需的文件将会传到这下面。
		"StartRunCmd": "",  // 每一个angelina-runner的启动命令，每个angelina-runner的启动命令需要一样。
		"ControllerContainer": "" // angelina-controller容器的名称
	}
（4）使用angelina -I 初始化 

	[root@kuber-master angelina]# angelina -I /tmp/angelina.json