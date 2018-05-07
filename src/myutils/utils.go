package myutils
import(
	"time"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"os"
	"path"
	"io"
	"crypto/sha256"
	"strings"
	"strconv"
	"compress/gzip"
)
func GetTime() string {
	timestamp := time.Now().Unix()
	tm := time.Unix(timestamp,0)
	return tm.Format("2006-01-02 15:04:05")
}
func Print(info,printStr string,exit bool) {
	fmt.Printf("%s\t%s\t%s\n",GetTime(),info,printStr)
	if exit == true {
		os.Exit(1)
	}
}
func GetSamplePrefix(name string) string {
	tmp := GetSha256(name)
	return "pipe" +  tmp[54:]
}
func GetSha256(name string) string {
	name = strings.Trim(name," ")
	name = strings.ToLower(name)
	hashObj := sha256.New()
	io.WriteString(hashObj,name)
	tmp := fmt.Sprintf("%x",hashObj.Sum(nil))
	return tmp
} 
func GetRunTime(startTime time.Time) string {
    now := time.Now()
    subM := now.Sub(startTime)
    hours := int(subM.Seconds()/3600)
    mins :=  int((subM.Seconds() - float64(hours * 3600))/60)
    mytime := float64(hours * 3600) + float64((mins) * 60)
    seconds := int(subM.Seconds() - mytime)
    return strconv.Itoa(hours) + "h " + strconv.Itoa(mins) +"m " + strconv.Itoa(seconds) +"s"

}
func GetRunTimeWithSeconds(tm int64) string {
	dur := float64(tm)
	hours := int(dur/3600)
	mins := int((dur - float64(hours * 3600))/60)
	mytime := float64(hours * 3600) + float64((mins) * 60)
	seconds := int(dur - mytime)
	return strconv.Itoa(hours) + "h " + strconv.Itoa(mins) +"m " + strconv.Itoa(seconds) +"s"
}
func GetOsEnv(env string) string {
	return os.Getenv(env)
}
func CopyFile(dstName,srcName string) {
	src,err := os.Open(srcName)
	defer src.Close()
	if err != nil {
		Print("Error","copy file failed,reason: " + err.Error(),true)
	}
	dst,err := os.OpenFile(dstName,os.O_WRONLY|os.O_CREATE,0644)
	defer dst.Close()
	if err != nil {
		Print("Error","copy file failed,reason: " + err.Error(),true)
	}
	_,err1 := io.Copy(dst,src)
	if err1 != nil {
		Print("Error","copy file failed,reason: " + err1.Error(),true)
		
	}
}
func TrimBase(srcName,filename string) string {
	data1 := strings.Split(filename,"/")
	data2 := strings.Split(srcName,"/")
	count := 0
	for i := 0;i < len(data2);i++ {
		if data1[i] == data2[i] {
			count++
		}
	}
	if count != 0 && data1[0] != ""{
		return strings.Join(data1[count:],"/")
	}else {
		return ""
	}
}

func CopyDirDFS(dstName,srcName string,ignore map[string]string) {
	os.MkdirAll(dstName,0755)
	dir_list,e := ioutil.ReadDir(srcName)
	if e != nil {
		fmt.Printf("read %s error,reason: %s\n",srcName,e.Error())
		os.Exit(2)
	}
	for _,v := range dir_list {
		if v.IsDir() {
			CopyDirDFS(path.Join(dstName,v.Name()),path.Join(srcName,v.Name()),ignore)
		}else {
			if _,ok := ignore[v.Name()]; !ok {
				if CheckFileExist(path.Join(dstName,v.Name())) {
					Print("Info","the file " + path.Join(dstName,v.Name()) + " has exist,skip to copy it to glusterfs.",false)
					continue
				}
				CopyFile(path.Join(dstName,v.Name()),path.Join(srcName,v.Name()))
				continue
			}
			if CheckFileExist(path.Join(dstName,ignore[v.Name()])) {
				Print("Info","the file " + path.Join(dstName,ignore[v.Name()]) + " has exist,skip to copy it to glusterfs.",false)
				continue
			}
			srcSuffix := strings.HasSuffix(v.Name(),".gz")
			dstSuffix := strings.HasSuffix(ignore[v.Name()],".gz")
			if srcSuffix == true && dstSuffix == false {
				GzipFile(path.Join(dstName,ignore[v.Name()]),path.Join(srcName,v.Name()))
			}else {
				CopyFile(path.Join(dstName,ignore[v.Name()]),path.Join(srcName,v.Name()))
			} 
		}
	}

}
func CopyDir(dstName,srcName string,ignore map[string]string) {
	os.MkdirAll(dstName,0777)
	filepath.Walk(srcName,func(filename string,fi os.FileInfo,err error) error {
		if filename != srcName {
			if err != nil {
				return err
			}
			if fi.IsDir() {
					os.MkdirAll(path.Join(dstName,fi.Name()),0777)
			}else {
				tfile := TrimBase(srcName,filename)
				newFile := path.Join(dstName,tfile)
				if len(ignore) != 0 {
					if _,ok := ignore[fi.Name()];!ok {
						if CheckFileExist(newFile) == false {
							CopyFile(newFile,filename)

						}else {
							Print("Info","the file " + newFile + " has exist,skip to copy it to glusterfs.",false)
						}
					}else {
						t := path.Base(filename)
						cfile := strings.Replace(newFile,t,ignore[fi.Name()],-1)
						srcSuffix := strings.HasSuffix(fi.Name(),".gz")
						dstSuffix := strings.HasSuffix(ignore[fi.Name()],".gz")
						if CheckFileExist(cfile) == false {
							if srcSuffix == true && dstSuffix == false {
								GzipFile(cfile,filename)
							}else {
								CopyFile(cfile,filename)
							}

						}else {
							Print("Info","the file " + cfile + " has exist,skip to copy it to glusterfs.",false)
						}	
					} 	
				}else {
						if CheckFileExist(newFile) == false {
							CopyFile(newFile,filename)
                            
                        }else {
                            Print("Info","the file " + newFile + " has exist,skip to copy it to glusterfs.",false)
                        } 
				}
			}
		}
		return nil
	})

}
func GzipFile(outfile,infile string) {
	gfile,err := os.Open(infile)
	defer gfile.Close()
	if err != nil {
		Print("Error","gzip file failed,reason: " + err.Error(),true)
	}
	content,err := gzip.NewReader(gfile)
	defer content.Close()
	if err != nil {
		Print("Error","gzip file failed,reason: "+ err.Error(),true)
	}	
	out,err := os.OpenFile(outfile,os.O_WRONLY | os.O_CREATE,0644)
	defer out.Close()
	if err != nil {
		Print("Error","gzip file failed,reason: " + err.Error(),true)
	}
	io.Copy(out,content)
}
func CheckFileExist(filename string) bool {
	var exist = true
	_,err := os.Stat(filename)
	if os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func WriteFile(filename,content string,toCreateFile bool) {
	if CheckFileExist(filename) && toCreateFile == true {
		file,err := os.Create(filename)
		defer file.Close()
		if err != nil {
			Print("Error","create file " + filename + " failed,reason: " + err.Error(),true)
		}
		_,err1 := io.WriteString(file,content)
		if err1 != nil {
			Print("Error","write file " + filename + " failed,reason: " + err1.Error(),true) 
		}
		file.Sync()
	} else if CheckFileExist(filename) && toCreateFile == false {
		file,err := os.OpenFile(filename,os.O_APPEND|os.O_WRONLY,os.ModeAppend)
		defer file.Close()
		if err != nil {
			Print("Error","write file " + filename + " failed,reason: " + err.Error(),true)
		}
		_,err1 := io.WriteString(file,content)
		if err1 != nil {
			Print("Error","write file " + filename + " failed,reason: " + err1.Error(),true)
		}
		file.Sync()
	} else {
		file,err := os.Create(filename)
		defer file.Close()
		if err != nil {
			Print("Error","write file " + filename + " failed,reason: " + err.Error(),true)
		}
		_,err1 := io.WriteString(file,content)
		if err1 != nil {
			Print("Error","write file " + filename + " failed,reason: " + err1.Error(),true)
		}
		file.Sync()

	}
}
