package client

var InitTemplate = `{
    "AuthFile": "",
    "ReferVolume": "refer-volume",
    "DataVolume" : "data-volume",
    "GlusterEndpoints": "gluster-cluster",
    "Namespace": "bio-system",
    "ScriptUrl": "",
    "OutputBaseDir": "",
	"StartRunCmd": "rundoc.sh",
	"ControllerServiceEntry": "angelina-controller:6300"
}`

var ConfigTemplate = `{
	"input-directory": "",
	"glusterfs-entry-directory": "",
	"sample-name": "",
	"template-env": [],
	"pipeline-template-name": "",
	"force-to-cover": "no"
}`

var PipelineTemplate = `{
	"pipeline-name": "",
	"pipeline-description": "",
	"pipeline-content": {
		"refer" : {
			"": "",
			"": ""
		},
		"input": [],
		"params": {
			"": "",
			"": ""
		},
		"resources-requests-1": {
			"cpu": "0m",
        	"memory":"0Mi"
		},
%s
	}
}`

var StepTemp = `		"%s": {
        	"pre-steps": [],
        	"container": "",
			"command-name": "",
        	"command": [],
        	"args":[],
        	"sub-args": [],
			"request-type": ""
		}%s
`
