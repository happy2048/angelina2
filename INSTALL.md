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
| 10.61.0.160  | glusterfs client, kubernetes master,kubernetes node  | kuber-master  |
| 10.61.0.161  |  glusterfs client,kubernetes node |  kuber-node1 |
| 10.61.0.162 |  glusterfs client,kubernetes node |  kuber-node2 |
| 10.61.0.163 |  glusterfs client,kubernetes node |  kuber-node3 |

（1） 在 vmnode1,vmnode2,vmnode3上执行如下步骤（我这只在vmnode1上执行，vmnode2,vmnode3执行命令一样）：

	[root@vmnode1 ~]# yum install -y glusterfs glusterfs-server glusterfs-fuse glusterfs-rdma

（2） 在vmnode1,vmnode2,vmnode3上启动glusterFS（我这只在vmnode1上执行，vmnode2,vmnode3执行命令一样）:

	[root@vmnode1 ~]# systemctl start glusterd
	[root@vmnode1 ~]# systemctl enable glusterd

（3） 在vmnode1上执行如下步骤，把vmnode2,vmnode3加入到集群当中：

	[root@vmnode1 ~]# gluster peer probe vmnode1
	[root@vmnode1 ~]# gluster peer probe vmnode2

（4） 在gluster1（vmnode1）上查看集群状态:

	[root@vmnode1 ~]# gluster peer status
	Number of Peers: 2

	Hostname: vmnode2
	Uuid: 3a6c9d1a-eb85-49e6-8a71-faf86b78d653
	State: Peer in Cluster (Connected)

	Hostname: vmnode3
	Uuid: 730ab35b-e2a3-4f7f-a718-5d9e1d7f49d9
	State: Peer in Cluster (Connected)

（5） 在vmnode1,vmnode2,vmnode3执行如下命令，创建数据存储目录（我这只在vmnode1上执行，vmnode2,vmnode3执行命令一样）：

	[root@vmnode1 ~]# mkdir  /mnt/gluster-data
	[root@vmnode1 ~]# mkdir  /mnt/gluster-refer
	[root@vmnode1 ~]# mkdir  /mnt/gluster-redis

（6） 创建volume,这里创建三个volume,一个是data-volume,一个是refer-volume,还有一个是redis-volume。data-volume是用来存放分析数据的，refer-volume是存放参考数据的，比如参考基因组文件，redis-volume是用来存放redis运行的数据，保证其持久化功能。下面命令只需在vmnode1上执行就行：

	[root@vmnode1 ~]# gluster volume create data-volume replica 3 vmnode1:/mnt/gluster-data  vmnode2:/mnt/gluster-data vmnode3:/mnt/gluster-data

	[root@vmnode1 ~]# gluster volume create refer-volume replica 3 vmnode1:/mnt/gluster-refer  vmnode2:/mnt/gluster-refer vmnode3:/mnt/gluster-refer

	[root@vmnode1 ~]# gluster volume create redis-volume replica 3 vmnode1:/mnt/gluster-redis  vmnode2:/mnt/gluster-redis vmnode3:/mnt/gluster-redis
	
（7） 在vmnode1上启动data-volume,refer-volume：

	[root@vmnode1 ~]# gluster volume start data-volume
	[root@vmnode1 ~]# gluster volume start refer-volume
	[root@vmnode1 ~]# gluster voluem start redis-volume
	
（8）在kuber-master,kuber-node1,kuber-node2,kuber-node3安装glusterfs客户端软件（整个kubernetes集群都需要安装glusterfs客户端软件），便于容器挂载（我这只是在kuber-master上执行如下命令，其他几点执行同样的命令）：

	[root@kuber-master ~]# yum install -y glusterfs-fuse glusterfs

**3.下载angelina源码**

（1）在kuber-master上使用git命令下载angelina源码：

	[root@kuber-master ~]# git clone https://github.com/happy2048/angelina2
	[root@kuber-master ~]# cd angelina2
（2）关于kubernetes中用到的yaml配置文件存放在config目录下

**4.创建namespace**

（1）为了不影响kubernetes中的其他应用，我们另外创建一个namespace,创建文件在下载的angelina2源码目录下的config目录下：

	[root@kuber-master config]# cat bio-system.yml
	apiVersion: v1
	kind: Namespace
	metadata:
       name: bio-system
       labels:
          name: bio-system

（2）这个文件不需要做任何修改

（3）在kuber-master上执行如下命令创建：

	[root@kuber-master config]# kubectl apply -f bio-system.yml

**5.在kubernetes创建gluster service和gluster endpoint**

（1）创建glusterfs service和glusterfs endpoint的yaml文件存放在angelina源码目录下的config目录，如下：

	[root@kuber-master config]# ll
	total 40
	-rw-r--r-- 1 root root  350 May  7 15:24 angelina-client-service-debug.yml
	-rw-r--r-- 1 root root  252 May  7 20:02 angelina-client-service.yml
	-rw-r--r-- 1 root root 1556 May  7 20:03 angelina-controller-deployment-debug.yml
	-rw-r--r-- 1 root root 1140 May  7 20:01 angelina-controller-deployment.yml
	-rw-r--r-- 1 root root  287 May  7 17:41 angelina-controller-service.yml
	-rw-r--r-- 1 root root   95 May  3 12:53 bio-system.yml
	-rw-r--r-- 1 root root  337 May  3 12:53 glusterfs-endpoints.json
	-rw-r--r-- 1 root root  183 May  3 12:53 glusterfs-service.json
	-rw-r--r-- 1 root root  712 May  7 19:54 redis-deployment.yml
	-rw-r--r-- 1 root root  210 May  7 19:59 redis-service.yml

（2）在kuber-master上创建gluster endpoints,这个文件不能直接运行，需要做修改，修改的部分是文件中“ip”域，这里根据自己实际情况填写自己的gluster server的地址，有多少个server就写多少ip（另外，需要记住我们这里创建endpoints名称为glusterfs-cluster,后面angelina初始化时需要用到）：

	[root@kuber-master config]# cat glusterfs-endpoints.json
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
			  "ip": "10.61.0.91"   // 这里的ip都需要修改
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
（3）使用如下命令创建：

	[root@kuber-master config]# kubectl apply -f glusterfs-endpoints.json

（4）在kuber-master上创建gluster service（这个文件不需要做修改）:

	[root@kuber-master config]# cat glusterfs-service.json 
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
	
（5）使用如下命令创建:

	[root@kuber-master config]# kubectl apply -f glusterfs-service.json

**6.在kubernetes上创建redis service和redis deployment**

（1）在kubernetes上创建redis deployment,如果glusterfs是按照前面的默认配置，那么这个文件不需要修改，直接运行即可，否则需要做如下修改（建议按默认配置）:

	a.glusterfs的域中的endpoins需要与前面创建的glusterfs endpoints名称一致

	b.namespace需要与前面创建的namespace一致

	c.redis挂载的卷需要同前面创建的redis存储的卷一致

	[root@kuber-master config]# cat redis-deployment.yml
	apiVersion: apps/v1beta1
	kind: Deployment
	metadata:
	  name: angelina-redis
	  namespace: bio-system
	  labels:
	    app: angelina-redis
	spec:
	  replicas: 1
	  selector:
	    matchLabels:
	       app: angelina-redis
	  template:
	    metadata:
	      labels:
	        app: angelina-redis
	    spec:
	      containers:
	      - name: angelina-redis
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
   
（2）使用如下命令创建：

	[root@kuber-master config]# kubectl apply -f redis-deployment.yml
	
（3） 在kubernetes上创建redis service（这里采用的service是kubernetes的nodePort的方式，另一种是LoadBalancer,这种方式会在每一个kubernetes节点上创建一个监听端口，访问任意一个节点的相应端口都可以访问redis,这里我们选择31000端口，这个需要记住，后面的angelina初始化需要用到）:

	[root@kuber-master config]# cat redis-service.yml 
	apiVersion: v1
	kind: Service
	metadata:
	  name: angelina-redis
	  namespace: bio-system
	spec:
	  type: NodePort
	  ports:
	  - port: 6380
	    targetPort: 6379
	    nodePort: 31000
	  selector:
	    app: angelina-redis

（4）使用如下命令创建：

	[root@kuber-master config]# kubectl apply -f redis-service.yml

**7.编译angelina client**

（1）在启动angelina controller之前，需要将angelina初始化信息存放到redis数据库中，当angelina controller启动时会读取相关的初始化信息，否则会报错，而初始化需要用到angelina client,所以需要先编译，也可以使用已经编译好的angelina client，放在源代码中的bin目录下。

（2）在编译之前，请确认golang是否安装。

（3）执行如下命令编译：

	[root@kuber-master angelina2]#  old=$(echo $GOPATH) && export GOPATH=$(pwd) && make && export GOPATH=$old 

（3）编译完成之后，会在当前目录下生成一个bin目录，如下:

	[root@kuber-master bin]# ll
	total 67240
	-rwxr-xr-x 1 root root 31925505 May  3 20:38 angelina
	-rwxr-xr-x 1 root root 31618999 May  3 20:38 angelina-controller
	-rwxr-xr-x 1 root root  5301877 May  3 20:38 angelina-runner
	
 这三个文件就是angelina的组件，其中的angelina就是angelina client。
 
**8.进行angelina初始化**

（1）初始化之前需要设置REDISADDR环境变量,我这里随便选择一个kuber-node1:31000

	[root@kuber-master  angelina2]# echo "export REDISADDR=kuber-node1:31000" >> /root/.bashrc
	[root@kuber-master  angelina2]# source  /root/.bashrc
	
（2）使用angelina  -g init 产生模板文件

	[root@kuber-master angelina2]# ./bin/angelina -g init 
	create init template file to /tmp/angelina.json

（3）编辑产生的模板文件
	
	[root@kuber-master angelina2]#  cat /tmp/angelina.json
	
	{
		"AuthFile": "/etc/kubernetes/admin.conf",  // kubernetes 认证文件路径，创建deployment时需要使用
		"ReferVolume": "refer-volume",  // referVolume是我们刚才创建的refer-volume，存放参考文件的volume
		"DataVolume" : "data-volume", // job使用的gluster volume，也就是我们刚才创建的data-volume
		"GlusterEndpoints": "glusterfs-cluster",// 我们创建glusterfs endpoint时使用的名称
		"Namespace": "bio-system", // 使用我们刚才创建的kubernetes namespace作为angelina专用namespace
		"ScriptUrl": "", //设置该选项的意义在于，如果我们做了很多关于angelina-runner的容器，如果现在我需要更新angelina-runner，那么这些容器需要从新制作，设置这个url，所有的容器的angelina-runner在运行时从这下载，然后运行,如果为空，默认从angelina controller处下载angelina-runner。
		"OutputBaseDir": "", // 在该物理机上glusterfs的data-volume挂载点，运行job所需的文件将会传到这下面。
		"StartRunCmd": "rundoc.sh",  // 每一个angelina-runner的启动命令，每个angelina-runner的启动命令需要一样,制作容器时需要将该脚本传入容器当中，否则运行肯定会失败。
		"ControllerServiceEntry": "angelina-controller:6300" // angelina-controler的访问端点，供angelina-runner使用，主要是对应后面创建的angelina controler的服务名称。
	}
（4）使用angelina -I 初始化 

	[root@kuber-master angelina]# angelina -I /tmp/angelina.json

**9.部署angelina controller**

（1）部署angelina controller所需的配置文件主要有config下的angelina-controller-deployment.yml，angelina-controller-service.yml，angelina-client-service.yml

（2）angelina-controller-deployment.yml的内容如下,如果采用默认配置，该文件不需要做任何修改即可运行：

	[root@kuber-master config]# cat angelina-controller-deployment.yml 
	apiVersion: apps/v1beta1
	kind: Deployment
	metadata:
	  name: angelina-controller
	  namespace: bio-system
	  labels:
	    app: angelina-controller
	spec:
	  replicas: 1
	  selector:
	    matchLabels:
	       app: angelina-controller
	  template:
	    metadata:
	      labels:
	        app: angelina-controller
	    spec:
	      containers:
	      - name: angelina-controller
	        image: happy365/angelina-controller:2.0
	        env:
	        - name: ANGELINA_REDIS_ADDR
	          value: angelina-redis
	        - name: ANGELINA_REDIS_PORT
	          value: "6380"
	        - name: ANGELINA_SERVER
	          value: ":6300"
	        ports:
	        - containerPort: 6300
	          protocol: UDP
	        - containerPort: 6300
	          protocol: TCP
	        volumeMounts:
	        - name: data
	          mountPath: /mnt/data
	        - name: refer
	          mountPath: /mnt/refer 
	      volumes:
	      - name: data
	        glusterfs:
	          endpoints: glusterfs-cluster
	          path: data-volume
	          readOnly: false   
	      - name: refer
	        glusterfs:
	          endpoints: glusterfs-cluster
	          path: refer-volume
	          readOnly: true

（3）使用如下命令创建：

	[root@kuber-master config]# kubectl apply -f angelina-controller-deployment.yml

（4）执行angelina-controller-service.yml，如果采用默认配置，改文件不需要做任何修改：

	[root@kuber-master config]# cat angelina-controller-service.yml 
	apiVersion: v1
	kind: Service
	metadata:
	  name: angelina-controller
	  namespace: bio-system
	spec:
	  ports:
	  - name: socket
	    port: 6300
	    protocol: UDP
	    targetPort: 6300
	  - name: http
	    port: 6300
	    protocol: TCP
	    targetPort: 6300
	  selector:
	    app: angelina-controller

（5）执行如下命令创建：

	[root@kuber-master config]# kubectl apply -f angelina-controller-service.yml

（6）执行angelina-client-service.yml，如果采用默认配置，不需要做任何修改（需要说明的是，这里仍然service仍然采用nodePort,后面的ANGELINA系统环境变量的设置只需要设置为任意一个节点的32000端口即可，例如: export ANGELINA=kuber-node1:32000）：

	[root@kuber-master config]# cat angelina-client-service.yml 
	apiVersion: v1
	kind: Service
	metadata:
	  name: angelina-client
	  namespace: bio-system
	spec:
	  type: NodePort
	  ports:
	  - name: restful
	    port: 6300
	    protocol: TCP
	    targetPort: 6300
	    nodePort: 32000
	  selector:
	    app: angelina-controller

（7）执行如下命令创建：
	
	[root@kuber-master config]# kubectl apply -f angelina-client-service.yml

（8）在angelina client所在物理机上设置ANGELINA系统环境变量，变量的值设置为任意的kuberntes node的32000端口即可：

	[root@kuber-master config]# echo "export ANGELINA=kuber-node1:32000" >> /root/.bashrc
	[root@kuber-master config]# source /root/.bashrc

**10.制作angelina runner容器**

（1） angelina runner是运行具体任务的容器，每个容器做的任务都不相同，在制作容器的的时候，需要在容器内容加入一个rundoc.sh的脚本文件，文件在angelina源码目录下的utils下，内容如下：

	[root@kuber-master angelina2]# cat utils/rundoc.sh 
	#!/bin/bash
	wget -c $SCRIPTURL -O /usr/bin/angelina-runner
	chmod +x /usr/bin/angelina-runner
	angelina-runner

（2）制作容器时，不要指定ENTRYPOINT,切记。

（3）以下是一个简单的bwa容器的例子，前面的内容不重要重要的是需要加入rundoc.sh这个脚本到容器中并赋予执行权限：

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
	ADD rundoc.sh  /usr/bin/rundoc.sh
	RUN chmod +x /usr/bin/rundoc.sh

（2）从上面的Dockerfile中可以看到，每一个任务容器都需要加入rundoc.sh这个工具。

（3）将制作好的容器上传到本地私有仓库。