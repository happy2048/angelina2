package cpfile
import(
	"myutils"
	"strings"
	"regexp"
	"io/ioutil"
    gjson "github.com/tidwall/gjson"
)
/*
func main() {
	data,_ := ioutil.ReadFile("mypipe.json")
	CopyFilesToGluster("/tmp/yang4","yang2",string(data))
}
*/
func CopyFilesToGluster(outdir,indir,data string)  {
	files := make(map[string]string)
	if ! gjson.Valid(data) {
        myutils.Print("Error","invalid json File",true)
    }
	
    jsonObj := gjson.Parse(data)
    jsonObj.ForEach(func(key,value gjson.Result) bool {
		if key.String() == "input" {
			if len(value.Array()) != 0 {
				for _,val := range value.Array() {
					if strings.Trim(val.String()," ") != "" {
						tstr := strings.Split(val.String(),"==>")
						if len(tstr) == 2 {
							src := strings.Trim(tstr[0]," ")
							dst := strings.Trim(tstr[1]," ")
							src,dst = LocateName(indir,src,dst)
							files[src] = dst
						}else {
							myutils.Print("Error","invalid mapping string " + val.String() + ",please to check the pipeline file",true)
						}
					}
				}
                	
			}
					
        }
		return true
    })
	myutils.CopyDirDFS(outdir,indir,files)
}

func LocateName(srcDir,patten1,patten2 string) (string,string) {
	srcName := ""
	dstName := patten2
	regConStar := regexp.MustCompile(`\*`)
	var leftStr string 	
	leftStr = strings.Replace(patten1,".","\\.",-1)
	leftStr = strings.Replace(leftStr,"*",".*",-1)
	leftStr = "\\b" + leftStr + "\\b"
	leftReg := regexp.MustCompile(leftStr)
	str := regConStar.FindAllString(patten2,-1)
	if len(str) != 0 {
		myutils.Print("Error","dist file name " + patten2 + " contains *,please give a ensure filename",true)
	}
	dirs,err := ioutil.ReadDir(srcDir)
	if err != nil {
		myutils.Print("Error","read directory " +srcDir + "failed",true)
		
	}
	for _,info := range dirs {
		if info.IsDir() == false {
			pats := leftReg.FindAllString(info.Name(),-1)
			if len(pats) != 0 {
				if srcName == "" {
					srcName = info.Name()
				}else {
					myutils.Print("Error","we checked that the " + srcDir + " has contains files like " + patten1 + " more than one,please remain one and try again",true)
				} 
			}
		}
	}
	if srcName == "" {
		myutils.Print("Error","we don't find the file like " + patten1 + ",please check the " + srcDir,true)
		
	}
	return srcName,dstName
}



