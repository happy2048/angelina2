package controller
import(
	"time"
	"strings"
)

func (ctrl *Controller) StatChangeChan() {
	for {
		select {
			case data := <- ctrl.FinishedSignal:
				go ctrl.HandleSignal(data)
		}
	}
}
func (ctrl *Controller) MyTickerFunc() {
	for {
		select {
			case <- ctrl.WriteLogTicker.C:
				ctrl.WriteLogs()
			case <- ctrl.LogTicker.C:
				ctrl.PrintInfo()
			case  <- ctrl.Ticker5.C:
				ctrl.RoundHandleJob()
				ctrl.CheckNameMap()
			case  <- ctrl.Ticker10.C:
				ctrl.FlashJobStepStatus()
			case <- ctrl.Ticker20.C:
				ctrl.PickJobStepToRun()
			case <- ctrl.Ticker30.C:
				ctrl.CheckJobStepAlive()
				ctrl.BackupJobs()
			case <- ctrl.Ticker60.C:
				ctrl.DeleteExpirationJob()
			case <- ctrl.HandleDataTicker.C:
				ctrl.RoundHandleRunnerData()
			
		}
	
	}

}
func (ctrl *Controller) HandleSignal(data string) {
	info := strings.Split(data,":")
	if len(info) == 2 {
		if info[1] == "deleting" {
			value := ctrl.JobsPool.Read(info[0])
			if value != nil {
				sample := value.SampleName
				ctrl.RunningJobs.Remove(sample)
				ctrl.DeletingJobs.Add(sample)
			}else {
				ctrl.AppendLogToQueue("Error","read",info[0],"failed,read it from JobsPool is nil")
			}
		}else if info[1] == "deleted" {
			value := ctrl.JobsPool.Read(info[0])
			if value != nil {
				sample := value.SampleName
				ctrl.DeletingJobs.Remove(sample)
				tdata := &SimpleJob{
					Name: sample,
					FinishedTime: time.Now(),
					Status: value.Status,
					Log: value.StepStatus}
				ctrl.FinishedJobs.Add(info[0],tdata)			
			}
		}
	}
}
