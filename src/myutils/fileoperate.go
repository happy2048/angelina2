package myutils
import(
	"fmt"
	"os"
	"io/ioutil"
	"path"
)

func PathOrFileExist(input string) (bool,error) {
	_,err := os.Stat(input) 
	if err == nil {
		return true,nil
	}
	if os.IsNotExist(err) {
		return false,nil
	}
	return false,err
}

func DeleteFilesFromDir(dir string) {
	_,err := os.Stat(dir)
	if err == nil {
		list,err1 := ioutil.ReadDir(dir)
		if err1 != nil {
			fmt.Println("read directory " + dir  + " error")
			return
		}
		for _,info := range list {
			if info.Size() == 0 {
				err2 := os.Remove(path.Join(dir,info.Name()))
				if err2 != nil {
					fmt.Println("remove file " + info.Name() + " failed")
					fmt.Println()

				}
			}
		}
	}
}

