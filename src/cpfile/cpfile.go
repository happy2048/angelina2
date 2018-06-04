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
	tfiles := make(map[string]string)
	dnames := make([]string,0,1000)
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
							dnames = append(dnames,dst)
							LocateName(indir,src,dst,tfiles)
						}else {
							myutils.Print("Error","invalid mapping string " + val.String() + ",please to check the pipeline file",true)
						}
					}
				}
                	
			}
					
        }
		return true
    })
	files := RevertName(tfiles)
	for _,dname := range dnames {
		if _,ok := files[dname]; !ok {
			myutils.Print("Warning","we don't find a file name in the input directory that match the target file name " + dname,false)
		}
	}
	myutils.CopyDirDFS(outdir,indir,tfiles)
}
func RevertName(first map[string]string) map[string]string {
	redata := make(map[string]string)
	for fkey,fvalue := range first {
		if _,ok := redata[fvalue]; ok {
			myutils.Print("Error","the input directory includes more than one files that match the filename " + fvalue,true)
		}else {
			redata[fvalue] = fkey
		}
	}
	return redata

}
func LocateName(srcDir,patten1,patten2 string,tfiles map[string]string) {
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
		myutils.Print("Error","read directory " +srcDir + " failed",true)
		
	}
	for _,info := range dirs {
		if info.IsDir() == false {
			pats := leftReg.FindAllString(info.Name(),-1)
			if len(pats) != 0 {
				tfiles[info.Name()] = patten2
			}
		}
	}
}



