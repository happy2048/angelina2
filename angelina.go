package main
import (
    "client"
)
func main() {
    cc:= client.NewConnector()
    cc.Start()
    rv := cc.ReturnInfo()
    bcli := client.NewBatchClient(rv.ControllerAddr,rv.RedisAddr,rv.GlusterEntryDir,rv.PipeTemp,rv.Force,rv.PipeTempName,rv.Tmp,rv.Env,rv.Names,rv.Inputs)
    bcli.Start()
}
