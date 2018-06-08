
**Angelina**

------------

Angelina: 一款开源的，适用于生物信息学分析的任务调度系统，基于kubernetes,redis,glusterfs和golang构建。

一个简单作业的的例子如下：

	1.一个作业分成了任务1，任务2，任务3，任务4
	2.执行的顺序是: 任务1 --> 任务2，任务3 --> 任务4
	3.任务2和任务3是并行执行

![](http://files.happy366.cn/files/images/task_sample.png)

angelina主要就是解决上面的任务执行顺序。

主要特点：
	
	1.每一个任务都是kubernetes中的一个deployment,只要做成相应的容器，无需重复部署任务所需的环境。
	2.支持状态记录，每一个任务运行成功以后都会记录其状态，在整个作业运行过程中，如果有任务运行失败，下次启动该作业，直接从错误任务重新开始运行。
	3.通过redis的订阅发布模式实现任务消息的接收与发送。
	4.通过glusterfs实现任务之间文件共享。
	5.可以把任务进一步拆成子任务。
	6.对运行的任务进行监控，出现错误可重新调度任务。
	7.当计算资源紧缺时，作业将排队等候，不会造成整个集群崩溃。
依赖：
	
	1.kubernetes (>= 1.8)
	2.glusterfs （>= 3.8）
	3.redis (>= 3.0)
	4.go (>= 1.8)
	

**安装**

参考文档： [angelina安装](https://github.com/happy2048/angelina2/blob/master/INSTALL.md)

**angelina架构图：**

（1）angelina架构图如下图所示：

![](http://files.happy366.cn/files/images/angelina-structure-0.png)
 
angelina controller架构图如下所示：

![](http://files.happy366.cn/files/images/angelina-structure-1.png)


（2）说明：

* 一个job有一个angelina controller与之对应，负责整个job的task调度，监控，错误恢复等操作。
* 一个job由很多个task组成，每一个task由一种runner与之对应，每一个runner是由一个deployment构成。
* 由angelina client去启动一个job。
* angelina client与angelina controller之间的通信以及angelina controller与angelina runner之间的通信是依靠redis的订阅发布模式。
* runner之间的文件共享通过glusterfs完成。
* task的运行状态存放在angelina-controller当中。

**angelina命令行帮助信息**
	
	[root@683ea81c73f6 biofile]# go run angelina.go -h
	Usage:
	  angelina [OPTIONS]
	
	Application Options:
	  -v, --version   software version.
                      打印版本信息
	  -f, --force     force to run all step of the sample,ignore they are succeed or failed last time.
                      是否覆盖上一次运行的状态，如果加上此选项，会重新运行所有任务，否则从上一次失败的任务开始运行。
	  -n, --name=     Sample name. 
                      此次运行的作业名称（也称一个sample）
	  -i, --input=    Input directory,which includes some files  that are important to run the sample.
                      运行作业所需一些文件的目录，需要将所需的文件放在该目录下，然后angelina会将该目录下的所有文件复制到glusterfs当中。
	  -o, --output=   Output directory,which is a glusterfs mount point,so that copy files to glusterfs.
                      gluterfs的data-volume的挂载点，执行作业所需文件需要复制到该目录下，运行作业完成的的结果也在该目录下。
	  -t, --template= Pipeline template name,the sample will be running by the pipeline template.
                      每一个作业运行都需要指定一个模板，该作业依据该模板运行。
	  -T, --tmp=      A temporary pipeline template file,defines the running steps,the sample will be
	                  running by it,can't be used with -t.
                      可以指定一个临时模板，需要提供临时模板文件，不能与-t一起使用。
	  -e, --env=      Pass variable to the pipeline template such as TEST="test",this option can be
	                  used many time,eg: -e TEST="test1" -e NAME="test".
                      动态的设置模板的参数，设置的值会覆盖模板中params域中的值。
	  -c, --config=   configure file,which include the values of -f -n -i -o -t.
                      可以将-f,-n,-i,-t所有参数写到一个配置文件中，配置文件的模板生成可以使用-g conf产生。
	  -a, --angelina= Angelina Controller address like ip:port,if you don't set this option,you must set the System Environment Variable ANGELINA.
                      设置angelina contoller的地址，格式为ip:port,如果加该选项，那么必须设置系统环境变量ANGELINA,否则程序不会运行。
	Batch Run Options:
	  -b, --batch     Start a batch run mode,this mode is only for job which includes pair-end fastq files.
                      使用batch run模式运行，这种模式仅适用于双端序列多样本情况，这种模式下，多个样本可以放在一起。	
	  -S, --split=    Split file name and use the first item of output array as job name. (default: _)
                      在batch run模式下，会以该分割符分割文件名，将分割数组中第一个元素作为job名。
	Other Options:
	  -p, --push=     Give a pipeline template file,and store it to angelina-controller.
                      如果有新的任务流模板，使用此选项把该模板保存到angelina-controller当中。
	  -l, --list      List the pipelines which have already existed.
                      列出当前已经存在的模板。
	  -D, --delete=   Delete the pipeline.
                      删除指定的模板名称。
	  -d, --del=      Given the job id or job name,Delete the job.
                      给出作业名或者作业号，删除指定的作业。
	  -j, --job=      Given the job id or job name,get the job status.
                      给出作业名或者作业号，获取其状态。
	  -J, --jobs      Get  all jobs status.
                      列出所有作业的状态。
	  -C, --cancel    Cancel send emails for current jobs who are watting to send email.
                      如果启用邮箱通知功能，使用该选项可以取消在该时间段应该发送的邮件。
	  -F, --flush     Cancel all jobs.
                      删除所有任务。
      -s, --step=     print the step run logs,must be used with -j.
                      打印指定job的指定step的日志信息，需跟-j一起使用。
	  -k, --keeping   Get the job status(or all jobs status) all the time,must along with -j or -J.
                      持续获取单个作业或者所有作业的状态，必须与-j或-J一起使用。
	  -q, --query=    give the pipeline id or pipeline name to get it's content.
                      查询指定模板的详细内容。
	  -g, --generate= Two value("conf","pipe") can be given,"pipe" is to generate a pipeline template file
	                  and you can edit it and use -s to store the pipeline,you can give value like "pipe:10"
	                  which "10" is represented total 10 steps;"conf" is to generate running configure file
	                  and you can edit it and use -c option to run the sample.
                      产生配置文件模板，任务流模板，其中配置文件模板供-c选项使用，任务流模板供-p使用。
	
	Help Options:
	  -h, --help      Show this help message
	



**angelina模板文件书写**

使用 angelina -g pipe 可以产生一个模板文件（使用 -g pipe:n 可以产生一个具有n个step的模板文件，比如： -g pipe:10），只需要在此基础上填写相应的内容即可，模板如下：

	{
		"pipeline-name": "",  // 模板名称
		"pipeline-description": "", // 模板描述
		"pipeline-content": {
			"refer" : {
				"": "",
				"": ""
			},
			"input": ["",""],
			"params": {
				"": "",
				"": ""
			},
			"step1": {
				"pre-steps": ["",""],
				"container": "",
				"command-name": "",
				"command": ["",""],
				"args":["",""],
				"sub-args": [""]
			},
			"step2": {
				"pre-steps": ["",""],
				"container": "",
				"command-name": "",
				"command": ["",""],
				"args":["",""],
				"sub-args": [""]
			}
		}
	}	


模板说明：
	
	（1） pipeline-name： 模板名称
	（2） pipeline-description： 模板描述
	（3） pipeline-content： 模板内容
	（4） pipeline-content: 主要分为五个域： refer,input,params,计算资源限制域，以及各个step,每个域都必须表示出来，如果没有数据就留空（计算资源域除外）。
refer域的说明：

	（1）主要在这设置一些任务所需的参考文件，比如参考基因组文件等，下面是个例子：
	
		"refer" : {
			"fasta": "reffa/b37/hg19.fasta",
			"dbsnp138": "refvcf/b37/dbsnp138.vcf"
		}
	（2）这个域所涉及的文件都是只读属性，也就是说你不可以在运行job当中去修改这些文件。
	（3）这个域中的文件路径是一个相对路径，主要是相对于之前我们配置的refer-volume，也就是说，假如我的refer-volume下面放了如下目录：
	
		[root@683ea81c73f6 refer]# ll
		total 17045972
		drwxr-xr-x 3 root root        4096 May  5  2017 annovar_db
		drwxr-xr-x 3 root root        4096 May  5  2017 reffa
		drwxr-xr-x 3 root root        4096 May  5  2017 refvcf
		-rw-r--r-- 1 root root 17455058559 Apr 25 02:39 test.tar.gz
		drwxr-xr-x 2 root root        4096 Apr  7 11:19 yang
	
	     如果我需要reffa/b37/hg19.fasta那么我只需要写reffa/b37/hg19.fasta就行，切记路径要写对，否则运行任务失败。
	（4）如果要在后续的step当中引用该域的一些文件，比如我需要hg19.fasta文件，只需要在step当中写成 “refer@fasta”就可以引用refer-volume下的reffa/b37/hg19.fasta文件。
	（5）如果该域没有内容，那么写成如下格式：
		
		"refer": {}
		
input域说明：
	
	（1） input域主要是对输入文件名称进行转换的，如果不转换，默认是原名复制，下面是一个例子：
	
		"input": [
			"*_R1.fastq.gz ==> test1_R1.fastq",
			"*_R2.fastq.gz ==> test1_R2.fastq",
			"a.txt ==> b.txt"
		]
	（2）上面的例子表达的意思是: 
		a.将input目录当中带有“_R1.fastq.gz”后缀的文件，复制到glusterfs中，并且解压缩成test1_R1.fastq(目前只支持gzip的解压缩)；
		b.将input目录当中带有“_R2.fastq.gz”后缀的文件，复制到glusterfs中，并且解压缩成test1_R2.fastq；
		c.将input目录当中的a.txt复制到glusterfs，并且重命名为b.txt
	（3） 该域中input目录下每一个匹配到的文件最多只能一个，例如“*_R1.fastq.gz ==> test1_R1.fastq”中，匹配到“*_R1.fastq.gz”的文件至多只有一个，假设在input目录当中有“test_R1.fastq.gz”和“test1_R1.fastq.gz”，将会报错，因为不知道将哪一个文件转化为"test1_R1.fastq"。
	（4）从input目录下复制的所有文件，将会存放在： glusterfs:data-volume/jobName/step0下 （data-volume是之前我们创建的job数据存放目录,jobName是每一个job的名称）
	
params域的说明：

	（1）params域主要是对step当中的命令的参数进行配置，与直接在step配置参数不同的是，该域的值可以在运行job时动态传入，下面是一个例子：
		"params": {
			"FASTQC": "2",
			"TRIM": "/root/Trimmomatic-0.36/trimmomatic-0.36.jar",
			"TRIMDIR":"/root/Trimmomatic-0.36"
		},
		比如上面的例子的当中，可以在命令行通过“-e  FASTQC=5”动态修改这个值。
	（2）在step当中引用params里面的值，比如在step当中需要使用“/root/Trimmomatic-0.36” 这个值，可以在step中使用“params@TRIMDIR”替换。

计算资源域说明：
	
	（1）计算资源域分为两种： requests和limits
	（2）requests表示容器运行需要的最低资源，如果集群剩余资源比最低资源还小，容器将不会调度，
	（3）limits表示容器运行最大可用的资源，如果容器运行时占用的资源比这个值大，容器将会被kill掉，不再运行。
	（4）requests资源域的键需要以"resources-requests-"开头，然后以1,2,3,4...依次定义，值中的cpu和memory可以不用全定义，例如：

		"resources-requests-1": {
            "cpu": "100m",
            "memory":"20Mi"
        }	
		"resources-requests-2": {
            "cpu": "300m",
            "memory":"5000Mi"
        }
		...
	（5）limits资源的键需要以"resources-limits-"开头，然后以1,2,3,4...依次定义,值中的cpu和memory可以不用全定义，例如：

		"resources-limits-1": {
            "cpu": "200m"
        },
	（6）cpu的单位为m，表示把一个cpu线程分成1000份,cpu: "300m"，表示0.3个cpu。
	（7）memory的单位为Mi,表示MB内存，200Mi,表示200MB内存
	（8）limits中的cpu值和memory值不能比requests中的cpu和memory值小，否则容器创建失败。
	（9）一般建议不要定义limits资源，因为对程序需要多少资源不熟悉，如果定义不合理，程序将永远不会运行成功，直接被kill掉。
	（10）后面的step域如果要引用该资源限制，可以在该域中加上如下语句：
	
		"limit-type":"resources-limits-1",
		"request-type":"resources-requests-2"
	(11) 没有资源限制可以不用定义，这是可选项。
		
step域说明：

	（1） step域是由众多的step组成，并且step编号必须从step1开始，连续不间断，不能重复定义，也就是说不能同时出现多个同样的step编号，下面是一个step例子：
		"step1": {
        	"pre-steps": [],
			"command-name":"fastqc",
        	"container": "registry.vega.com:5000/fastqc:1.0",
        	"command": ["fastqc"],
        	"args":["-o step1@","-f fastq","step0@test1_R1.fastq step0@test1_R2.fastq"],
        	"sub-args": [],
			"request-type":"resources-requests-2", 
			"limit": ["300m","100Mi"]
		}
		pre-steps: 该step所依赖的step,有多少写多少，没有就写成[]。
		command-name: 为该step运行的命令取一个别名，不能留空。
		container： 运行该step所需要的容器，不能留空。
		command: 该step所需要运行的命令，数组内容会拼接成字符串，不能留空。
		args: 命令所需的参数，数组内容会拼接成字符串，不能留空。
		sub-args: 数组类型，数组的长度代表在该step需要启动多少个这样的容器，来处理不同输入不同输出，举个例子，如果sub-args数组为["a.out","b.out"],那么该step总共需要启动两个容器，第一个容器处理的命令是command + args + sub-args[0],第二个容器处理的命令是command + args + sub-args[1]，这样设计的目的是可让angelina具有split-merge功能，不过merge得自行处理。
		request-type(或者limit-type): 字符串，使用上面定义的资源域中的值。
		limit(或者request): 不用上面资源域定义的资源限制，直接定义资源限制，数组类型，第一个值为cpu使用量，第二个值为内存使用量。不能与limit-type(或者request-type)同时使用。
	（2） 下面是一个启动多个相同step的例子：
	
		"step2": {
        	"pre-steps": ["step1"],
			"command-name":"test",
        	"container": "registry.vega.com:5000/test:1.0",
        	"command": ["/bin/bash","/root/test.sh"],
        	"args":["name","30"],
        	"sub-args": ["a.out","b.out"]
		}
		angelina会启动两个registry.vega.com:5000/test:1.0 类型的容器来运行step2，第一个容器运行的命令是：“/bin/bash  /root/test.sh name 30 a.out”,第二个容器运行的命令是“/bin/bash /root/test.sh name 30 b.out”
		启动容器的数量有sub-args数组长度确定。
	（3）如果该step只需要运行一个命令，那么只需要将sub-args留空就行，那么运行的命令就是command + args。
	（4）如果在该step当中需要引用pre-steps当中的一些文件，可以使用pre-step的名称+“@”来实现，例如下面：
		"step2": {
        	"pre-steps": ["step1"],
			"command-name":"test",
        	"container": "registry.vega.com:5000/test:1.0",
        	"command": ["/bin/bash","/root/test.sh"],
        	"args":["name","30"，"step1@my.txt","refer@fasta","paramas@TRIMDIR"],
        	"sub-args": ["a.out","b.out"]
		}
		step2用到了step1的my.txt，只需要使用step1@my.txt就行。
	（5）在step当中用到的所有文件都是使用相对路径。
	（6）step0只能被引用，不能被定义,否则模板校验不会通过。
	（7）除了request-type（或者limit-type）和 request(或者limit)可以不定义外，其他都必须填写，没有用相应的空值替代。

**命令行使用**

1.查看angelina上运行的所有job：

	[root@kuber-master ~]# angelina -J
                                	Angelina                                    
	********************************************************************************
	Date       Time       Job Id           Status         Job Name
	--------------------------------------------------------------------------------
	2018-06-08 17:26:23   pipe3c7113f380   Running        1701239-M
	********************************************************************************

2.查看指定job的运行状态：

	[root@kuber-master ~]# angelina -j  1701239-M 
	                                                      Running  Status                                             
	*******************************************************************************************************************************
	Software          Name: angelina
	Software       Version: v2.3
	Template          Name: ExonAnalysis
	Template Estimate Time: 0h 0m 0s
	Running Sample    Name: 1701239-M
	Already Running   Time: 48h 7m 24s
	------------------------------------------------------------------------------------------------------------------------------
	Date       Time       Step     Sub  Status   Deployment-Id    Run-Time     Pre-Steps             Command                  
	------------------------------------------------------------------------------------------------------------------------------
	2018-06-08 17:27:43   step1    0    succeed  deploy6e9ac4b8b  0h 0m 0s     ---                   Fastp                    
	2018-06-08 17:27:43   step2    0    succeed  deploye17db9beb  0h 0m 0s     1                     Minimap2                 
	2018-06-08 17:27:43   step3    0    succeed  deploya9d033403  5h 17m 40s   20                    Sambamba View            
	2018-06-08 17:27:43   step4    0    succeed  deploy4ea00ccbb  1h 42m 26s   3                     Sambamba Markdup         
	2018-06-08 17:27:43   step5    0    succeed  deploy5d22713c1  0h 20m 50s   4                     RealignerTargetCreator   
	2018-06-08 17:27:43   step6    0    succeed  deploy2cf0e6c36  3h 1m 50s    5                     IndelRealigner           
	2018-06-08 17:27:43   step7    0    succeed  deploy16d634e80  7h 19m 50s   6                     BaseRecalibrator         
	2018-06-08 17:27:43   step8    0    succeed  deploy99ea10261  8h 46m 49s   7                     PrintReads               
	2018-06-08 17:27:43   step9    0    running  deploy3fc36773a  21h 36m 49s  8                     HaplotypeCaller          
	2018-06-08 17:27:43   step10   0    ready    not allocate     0h 0m 0s     9                     SelectVariants           
	2018-06-08 17:27:43   step11   0    ready    not allocate     0h 0m 0s     10                    VariantFiltration        
	2018-06-08 17:27:43   step12   0    ready    not allocate     0h 0m 0s     9                     SelectVariants           
	2018-06-08 17:27:43   step13   0    ready    not allocate     0h 0m 0s     12                    VariantFiltration        
	2018-06-08 17:27:43   step14   0    ready    not allocate     0h 0m 0s     13,11                 CombineVariants          
	2018-06-08 17:27:43   step15   0    ready    not allocate     0h 0m 0s     14                    SelectVariants           
	2018-06-08 17:27:43   step16   0    ready    not allocate     0h 0m 0s     15                    Annovar                  
	2018-06-08 17:27:43   step17   0    ready    not allocate     0h 0m 0s     15                    SnpEff                   
	2018-06-08 17:27:43   step18   0    ready    not allocate     0h 0m 0s     17,16                 ProduceJson              
	2018-06-08 17:27:43   step19   0    ready    not allocate     0h 0m 0s     18                    PushJson                 
	2018-06-08 17:27:43   step20   0    succeed  deploy242f8522c  0h 0m 0s     2                     Samblaster               
	*******************************************************************************************************************************

3.查看指定job的相关step日志信息：

	[root@kuber-master ~]# angelina -j  1701239-M  -s step3
	2018-06-06 14:37:43	Info	the command sambamba view -S -f bam -l 0  /mnt/data/1701239-M/step20/step20_samblaster.sam  | sambamba sort -t 6 -m 40G  --tmpdir /mnt/data/1701239-M/step3/tmp   -o /mnt/data/1701239-M/step3/step3_sorted.bam  /dev/stdin &&   sambamba index /mnt/data/1701239-M/step3/step3_sorted.bam  will run
	2018-06-06 14:37:43	Info	the command run status sambamba view -S -f bam -l 0  /mnt/data/1701239-M/step20/step20_samblaster.sam  | sambamba sort -t 6 -m 40G  --tmpdir /mnt/data/1701239-M/step3/tmp   -o /mnt/data/1701239-M/step3/step3_sorted.bam  /dev/stdin &&   sambamba index /mnt/data/1701239-M/step3/step3_sorted.bam  has send to channel

4.连续侦听指定job的运行状态：

	[root@kuber-master ~]# angelina -j  1701239-M -k

5.连续侦听所有job的运行状态：

	[root@kuber-master ~]# angelina -J -k

6.创建一个job：
	
	[root@kuber-master ~]# angelina -n jobName -t template -i inputDir -o glusterfsVolumeMountPoint

7.使用配置文件创建job（config.json由angelina -g conf产生）：

	[root@kuber-master ~]# angelina -c config.json

8.保存一个新的任务流模板：

	[root@kuber-master ~]# angelina -p pipeline.json

9.在创建job时，使用临时模板，不使用已有模板，需要提供临时模板文件：

	[root@kuber-master ~]# angelina -n jobName -t tmpPipeline.json  -i inputDir -o glusterfsVolumeMountPoint

10.删除指定job：

	[root@kuber-master ~]# angelina -d jobName

11.删除所有job：

	[root@kuber-master ~]# angelina -F

12.使用batch run模式运行job：

	[root@kuber-master ~]# angelina -n jobName -t template  -i inputDir -o glusterfsVolumeMountPoint -b -S "_"
 
 inputDir的内容可以像下面这种样子，多个样本放在一起：

	[root@virt2 fqfile]# ll
	total 39060
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui1_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:52 mahui1_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui2_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui2_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui3_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui3_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui4_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui4_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui5_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui5_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui6_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui6_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:51 mahui7_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:51 mahui7_test_R2.fastq.gz
	-rw-r--r-- 1 root root 2178112 Jun  3 18:52 mahui8_test_R1.fq.gz
	-rw-r--r-- 1 root root 2261739 Jun  3 18:52 mahui8_test_R2.fq.gz
	-rw-r--r-- 1 root root 2178112 May  2 10:22 mahui_test_R1.fastq.gz
	-rw-r--r-- 1 root root 2261739 May  2 10:22 mahui_test_R2.fastq.gz
	-rw-r--r-- 1 root root       0 Jun  3 19:59 yang1

13.给模板参数运行时动态传入值，如果在模板的params域有一个key为THREAD,制作模板的时候设置值为2，那么我在运行job时可以做如下修改：

	[root@kuber-master ~]# angelina -n jobName -t template  -i inputDir -o glusterfsVolumeMountPoint  -e THREAD=5

14.当在运行job时，某个step运行失败，如果再次运行该job，只会从运行失败的step开始运行，如果需要从第一步重新运行，使用-f选项：

	[root@kuber-master ~]# angelina -n jobName -t template  -i inputDir -o glusterfsVolumeMountPoint -f

15.如果启用邮件通知功能，每隔一段时间（这个时间由用户指定），会向事先定义好的邮箱通知在该时间段之内运行完成的任务的状态（成功或者失败），使用-C可以取消在该时间段内需要发送邮件的job：

	[root@kuber-master ~]# angelina -C

**一个简单的模板例子**
	
	
	{
		"refer" : {
			"fasta": "reffa/b37/human_g1k_v37_decoy.fasta"
		},
		"input": ["*_R1.fastq.gz ==> test1_R1.fastq","*_R2.fastq.gz ==> test1_R2.fastq"],
		"params": {
			"FASTQC": "2",
			"TRIM": "/root/Trimmomatic-0.36/trimmomatic-0.36.jar",
			"TRIMDIR":"/root/Trimmomatic-0.36"
		},
		"resources-limits-1": {
            "cpu": "200m"
        },
        "resources-requests-1": {
            "cpu": "100m",
            "memory":"20Mi"
        },
		"step1": {
        	"pre-steps": [],
			"command-name":"fastqc",
        	"container": "registry.vega.com:5000/fastqc:1.0",
        	"command": ["fastqc"],
        	"args":[
				"-t params@FASTQC",
				"-o step1@",
				"-f fastq",
				"step0@test1_R1.fastq step0@test1_R2.fastq"
			],
        	"sub-args": [],
			"request-type": "resources-requests-1"
		},
		"step2": {
        	"pre-steps": [],
			"command-name": "trimmomatic-0.36.jar",
        	"container": "registry.vega.com:5000/trim:1.0",
        	"command": ["java","-jar","params@TRIM"],
        	"args":[
				"PE -phred33",
				"-threads 2",
				"step0@test1_R1.fastq step0@test1_R2.fastq step2@test1_R1_paired.fastq step2@test1_R1_unpaired.fastq step2@test1_R2_paired.fastq step2@test1_R2_unpaired.fastq",
				"LEADING:3 TRAILING:3 SLIDINGWINDOW:4:15 MINLEN:75",
				"ILLUMINACLIP:params@TRIMDIR/adapters/TruSeq3-PE-2.fa:2:30:10"
			],
        	"sub-args": [],
			"request-type": "resources-requests-1",
			"limit-type": "resources-limit-1"
		},
		"step3": {
			"pre-steps":["step2"],
			"command-name":"bwa mem",
			"container": "registry.vega.com:5000/bwa:1.0",
			"command": ["bwa","mem"],
			"args":[
				"-t 1",
				"-M",
				"-R '@RG\\tID:ST_Test_Yang_329_H7NNYALXX_6\\tSM:ST_Test_Liuhong\\tLB:WBJPE171539-01\\tPU:H7NNYALXX_6\\tPL:illumina\\tCN:thorgene'",
				"refer@fasta"
			],
			"sub-args":[
				"step2@test1_R1_paired.fastq step2@test1_R2_paired.fastq > step3@test1.sam",
				"step2@test1_R1_paired.fastq step2@test1_R2_paired.fastq > step3@test2.sam"
			],
			"request":["30m","100Mi"],
			"limit": ["100m","300Mi"]
		}
	}
模板会自动转化成如下模板，所以不需要写文件的绝对路径（这个例子中job名为mahui,data-volume会被挂载到容器的/mnt/data,refer-volume会被挂载到容器的/mnt/refer）：

	{
		"step1":{
			"Command":"fastqc   -t 2  "
			"CommandName":"fastqc",
			"Args":"-t 2 -o /mnt/data/mahui/step1/  -f fastq  /mnt/data/mahui/step0/test1_R1.fastq /mnt/data/mahui/step0/test1_R2.fastq ",
			"Container":"registry.vega.com:5000/fastqc:1.0",
			"Prestep":[],
			"SubArgs":[],
			"ResourcesRequests":["100m","20Mi"]
		},
		"step2":{
			"Command":"java -jar  /root/Trimmomatic-0.36/trimmomatic-0.36.jar  ",
			"CommandName":"trimmomatic-0.36.jar",
			"Args":"PE -phred33 -threads 2  /mnt/data/mahui/step0/test1_R1.fastq /mnt/data/mahui/step0/test1_R2.fastq /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R1_unpaired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq /mnt/data/mahui/step2/test1_R2_unpaired.fastq  LEADING:3 TRAILING:3 SLIDINGWINDOW:4:15 MINLEN:75 ILLUMINACLIP:/root/Trimmomatic-0.36/adapters/TruSeq3-PE-2.fa:2:30:10",
			"Container":"registry.vega.com:5000/trim:1.0",
			"Prestep":[],
			"SubArgs":[],
			"ResourcesLimits":["200m",""],
			"ResourcesRequests":["100m","20Mi"]
		},
		"step3":{
			"Command":"bwa mem",
			"CommandName":"bwa mem",
			"Args":"-t 1 -M -R '@RG\\tID:ST_Test_Yang_329_H7NNYALXX_6\\tSM:ST_Test_Liuhong\\tLB:WBJPE171539-01\\tPU:H7NNYALXX_6\\tPL:illumina\\tCN:thorgene'  /mnt/refer/reffa/b37/human_g1k_v37_decoy.fasta  ",
			"Container":"registry.vega.com:5000/bwa:1.0",
			"Prestep":["step2"],
			"SubArgs":[
				" /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq > /mnt/data/mahui/step3/test1.sam ",
				" /mnt/data/mahui/step2/test1_R1_paired.fastq /mnt/data/mahui/step2/test1_R2_paired.fastq > /mnt/data/mahui/step3/test2.sam "
			],
			"ResourcesRequests":["30m","100Mi"],
			"ResourcesLimits":["100m","300Mi"]
		}
	}

使用方法：

1.为了做测试我做了一个测试容器，这个容器有一个test.sh脚本，脚本内容如下：

	#!/bin/bash
	status=$1
	mysleep=$2
	infile=$3
	outfile=$4
	echo "command start to run"
	echo "sleep $mysleep seconds"
	sleep $mysleep
	if ! cat $infile &> /dev/null;then
	    exit 1
	fi
	echo ${DEPLOYMENTID}" run me" > $outfile
	echo "command run finished"
	if [ $status == "succeed" ];then
	    exit 0
	else
	    exit 1
	fi

	接收4个参数：
	（1）status: 命令最后是运行成功还是失败，succeed表示成功,failed表示失败
	（2）mysleep: sleep多少秒钟，用来模拟容器需要运行多长时间
	（3）infile: 用来模拟运行该容器需要的输入文件。
	（4）outfile: 用来模拟容器的输出文件

2.定义一个pipeline.json,内容如下：

	{
		"pipeline-name": "test",
		"pipeline-description": "test  pipeline",
		"pipeline-content": {
			"input": [
				"entry-file-1.txt ==> step0-out-1.txt",
				"entry-file-2.txt ==> step0-out-2.txt",
				"*_other.txt  ==> step0-other-out.txt"
			],
			"params": {
				"STEP2-STATUS": "succeed",
				"STATUS": "succeed",
				"SLEEP": "80"
			},
			"resources-requests-1": {
				"cpu":"500m",
				"memory": "600Mi"
			},
			"resources-requests-2": {
				"cpu":"600m",
				"memory":"1000Mi"
			},
			"step1": {
				"pre-steps": [],
				"command-name":"step1-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["params@STATUS","params@SLEEP","step0@step0-out-1.txt","step1@step1-out.txt"],
	        	"sub-args": [],
				"request-type": "resources-requests-1"
			},
			"step2": {
				"pre-steps": [],
				"command-name":"step2-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["succeed","50","step0@step0-out-2.txt","step2@step2-out.txt"],
	        	"sub-args": [],
				"request-type": "resources-requests-1"
			},
			"step3": {
				"pre-steps": ["step1","step2"],
				"command-name":"step3-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["params@STATUS","params@SLEEP","step0@step0-out-1.txt","step3@step3-out.txt"],
	        	"sub-args": [],
				"request": ["500m","500Mi"] 
			},
			"step4": {
				"pre-steps": ["step2","step3"],
				"command-name":"step4-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["succeed","70","step2@step2-out.txt","step4@step4-out.txt"],
	        	"sub-args": [],
				"request-type": "resources-requests-2"
			},
			"step5": {
				"pre-steps": ["step4","step3"],
				"command-name":"step5-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["succeed","params@SLEEP","step4@step4-out.txt"],
	        	"sub-args": ["step5@step5-out.txt","step5@step5-out-1.txt"],
				"request": ["800m","900Mi"]
			},
			"step6": {
				"pre-steps": ["step5"],
				"command-name":"step6-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["failed","20","step5@step5-out.txt"],
	        	"sub-args": ["step6@step6-out.txt","step6@step6-out-1.txt","step6@step6-out-3.txt"],
				"request-type": "resources-requests-1"
			},
			"step7": {
				"pre-steps": ["step6"],
				"command-name":"step7-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["succeed","10","step6@step6-out.txt","step7@step7-out.txt"],
	        	"sub-args": [],
				"request-type": "resources-requests-1"
			},
			"step8": {
				"pre-steps": ["step6"],
				"command-name":"step8-cmd",
	        	"container": "happy365/angelina-test:2.0",
	        	"command": ["test.sh"],
	        	"args":["succeed","params@SLEEP","step6@step6-out-1.txt","step8@step8-out.txt"],
	        	"sub-args": [],
				"request": ["400m","500Mi"]
			}
		
		}
	}

3.初始化模板：

	[root@kuber-master docker]# angelina -p pipeline.json

	在初始化过程中，会对模板进行严格校验，请按照模板要求填写：
	
	如果出现以下错误，表名该文件不符合json文件的格式：
	
	invalid pipeline file,parse failed,some commas are add in bad area or don't delete the annotation?

4.创建输入目录,并包括以下文件，文件内容随意：
	
	[root@kuber-master sample]# ll input/
	total 2
	-rw-r--r-- 1 root root 19 May 10 14:11 entry-file-1.txt
	-rw-r--r-- 1 root root 20 May 10 14:11 entry-file-2.txt
	-rw-r--r-- 1 root root 12 May 10 14:13 test2_other.txt

5.创建一个配置文件，使用如下命令产生模板：

	[root@kuber-master sample]# angelina -g conf	
    [root@kuber-master sample]# cat config.json
	{
		"input-directory": "/root/biofile/sample/input", //输入目录
		"glusterfs-entry-directory": "/mnt/data",  // glusterfs data-volume的挂载点
		"sample-name": "test",  //作业名
		"template-env": ["REDIS=33","YANG=33"],  // 模板的params参数，在这里可以动态传入
		"pipeline-template-name": "test", // 模板名称
		"force-to-cover": "yes" // 是否强制覆盖上次的内容
	}

6.创建作业：

	[root@kuber-master sample]# angelina -c config.json

7.查看作业运行情况：

	[root@kuber-master sample]# angelina -j test 

	                                                      Running  Status                                             
	*******************************************************************************************************************************
	Software          Name: angelina
	Software       Version: v2.3
	Template          Name: test
	Template Estimate Time: 0h 0m 0s
	Running Sample    Name: test
	Already Running   Time: 0h 8m 14s
	------------------------------------------------------------------------------------------------------------------------------
	Date       Time       Step     Sub  Status   Deployment-Id    Run-Time     Pre-Steps             Command                  
	------------------------------------------------------------------------------------------------------------------------------
	2018-05-10 14:23:06   step1    0    ready    not allocate     0h 0m 0s     ---                   step1-cmd                
	2018-05-10 14:23:06   step2    0    ready    not allocate     0h 0m 0s     ---                   step2-cmd                
	2018-05-10 14:23:06   step3    0    ready    not allocate     0h 0m 0s     1,2                   step3-cmd                
	2018-05-10 14:23:06   step4    0    ready    not allocate     0h 0m 0s     2,3                   step4-cmd                
	2018-05-10 14:23:06   step5    0    ready    not allocate     0h 0m 0s     4,3                   step5-cmd                
	2018-05-10 14:23:06   step5    1    ready    not allocate     0h 0m 0s     4,3                   step5-cmd                
	2018-05-10 14:23:06   step6    0    ready    not allocate     0h 0m 0s     5                     step6-cmd                
	2018-05-10 14:23:06   step6    1    ready    not allocate     0h 0m 0s     5                     step6-cmd                
	2018-05-10 14:23:06   step6    2    ready    not allocate     0h 0m 0s     5                     step6-cmd                
	2018-05-10 14:23:06   step7    0    ready    not allocate     0h 0m 0s     6                     step7-cmd                
	2018-05-10 14:23:06   step8    0    ready    not allocate     0h 0m 0s     6                     step8-cmd                
	*******************************************************************************************************************************

	加上 -k 选项可以持续查看

8.查看整个系统的作业运行情况：

	[root@kuber-master sample]# angelina -J


	                                Angelina                                    
	********************************************************************************
	Date       Time       Job Id           Status         Job Name
	--------------------------------------------------------------------------------
	2018-05-10 14:42:16   pipe15b0f00a08   Running        test
	2018-05-10 14:42:16   pipe1344e3aab5   Finished       yang80
	2018-05-10 14:42:16   pipef7f5f4583c   Finished       yang86
	2018-05-10 14:42:16   pipe5d61494236   Finished       yang19
	2018-05-10 14:42:16   pipe7e7cf0b5c9   Finished       yang25
	2018-05-10 14:42:16   pipea790286f70   Finished       yang26
	2018-05-10 14:42:16   pipeead06fecd2   Finished       yang15
	2018-05-10 14:42:16   pipe416798286b   Finished       yang16
	2018-05-10 14:42:16   pipecef04dd8f7   Finished       yang38
	2018-05-10 14:42:16   piped0d60612c8   Finished       yang68
	2018-05-10 14:42:16   pipeb788eabf52   Finished       yang90
	2018-05-10 14:42:16   pipe57117abcce   Finished       yang2
	2018-05-10 14:42:16   piped61c7326ed   Finished       yang5
	********************************************************************************

9.删除指定作业：
 
 	[root@kuber-master sample]# angelina -d test
	[root@kuber-master sample]# angelina -J
	                                   Angelina                                    
	********************************************************************************
	Date       Time       Job Id           Status         Job Name
	--------------------------------------------------------------------------------
	2018-05-10 14:44:50   pipe15b0f00a08   Deleting       test
	2018-05-10 14:44:50   pipe6189bf0463   Finished       yang76
	2018-05-10 14:44:50   pipe71e47a4b1d   Finished       yang88
	....
	....
	
	********************************************************************************


