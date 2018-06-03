package client
import(
	"os"
	"strings"
	"io/ioutil"
	"myutils"
	"path"
	"fmt"
)
func (cc *Connector) PreBatchRun() []string {
	indir := cc.Rv.Input
	split := cc.Opt.BatchRun.Split
	info,err := os.Stat(indir)
    if os.IsNotExist(err) {
		fmt.Println("Error: the input directory does not exist,exit.")
		os.Exit(4)
    }else if err != nil {
		fmt.Println("Error: check input directory failed,reason: ",err.Error())
		os.Exit(4)
	}
	if ! info.IsDir() {
		fmt.Println("Error: we check that input directory is a file,exit")
		os.Exit(4)
	}
	files,err := ioutil.ReadDir(indir)
	if err != nil {
		fmt.Println("Error: read input directory failed,reason: ",err.Error())
		os.Exit(4)
	}
	fastqs := make([]string,0,2000)
	hasReadMap := make(map[string]bool)
	otherDirMap := make(map[string]bool)
	otherMap := make(map[string]bool)
	batchDirs := make([]string,0,2000)
	for _,file := range files {
		if file.IsDir() {
			if strings.Index(file.Name(),"batch_") != 0 {
				otherDirMap[file.Name()] = true
			} 
			continue
		}
		if strings.HasSuffix(file.Name(),"fastq") || strings.HasSuffix(file.Name(),"fastq.gz") || strings.HasSuffix(file.Name(),"fq") || strings.HasSuffix(file.Name(),"fq.gz") {
			fastqs = append(fastqs,file.Name())
		}else {
			otherMap[file.Name()] = true
		}
				
	}
	for _,key := range fastqs {
		if _,ok := hasReadMap[key]; ok {
			continue
		}
		matchLen := 0
		matchStr := ""
		for _,ikey := range fastqs {
			if ikey == key {
				continue
			}
			tlen := MatchMaxLen(key,ikey)
			if tlen > matchLen {
				matchLen = tlen
				matchStr = ikey
			}
		}
		if IsMatch(key,matchStr) {
			err := CreateBatchDir(indir,key,matchStr,split)
			if err != nil {
				fmt.Printf("Error: move files %s and %s failed,reason: %s\n",key,matchStr,err.Error())
				os.Exit(4)
			}
			hasReadMap[key] = true
			hasReadMap[matchStr] = true
		}
	}
	filesList,err := ioutil.ReadDir(indir)
	for _,file := range filesList {
		if file.IsDir() && strings.Index(file.Name(),"batch_") == 0 {
			batchDirs = append(batchDirs,path.Join(indir,file.Name()))
		}
	}
		
	for key,_ := range otherMap {
		for _,bdir := range batchDirs {
			myutils.CopyFile(path.Join(bdir,key),path.Join(indir,key))
		}
	}
	for key,_ := range otherDirMap {
		for _,bdir := range batchDirs {
			myutils.CopyDir(path.Join(bdir,key),path.Join(indir,key))
		}
	
	}
	return batchDirs

}
func MatchMaxLen(str1,str2 string) int {
	if len(str1) != len(str2) {
		return 0
	}
	count := 0
	for i := 0;i < len(str1); i++ {
		if str1[i] != str2[i] {
			return count
		}
		count++
	}
	return count

}
func IsMatch(str1,str2 string) bool {
	if len(str1) != len(str2) {
		return false
	}
	for i := 0;i < len(str1); i++  {
		if str1[i] != str2 [i]  && str1[i] + str2[i] != 99 {
			return false
		}
	}
	return true
}
func CreateBatchDir(indir,str1,str2,split string) (error) {
	name1 := strings.Split(str1,split)[0]
	name2 := strings.Split(str2,split)[0]
	if name1 != name2 {
		fmt.Printf("the split string is invalid for %s and %s,exit.\n",str1,str2)
		os.Exit(4)
	}
	batchDir := path.Join(indir,"batch_" + name1)
	os.MkdirAll(batchDir,0755)
	err := os.Rename(path.Join(indir,str1),path.Join(batchDir,str1))
	if err != nil {
		return err
	}
	err = os.Rename(path.Join(indir,str2),path.Join(batchDir,str2))
	return err
}

