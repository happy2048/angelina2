package main
import (
	"client"
)
func main() {
	cc:= client.NewConnector()
    cc.Start()
	rv := cc.ReturnInfo()
    cli := client.NewClient(rv.ControllerAddr,rv.RedisAddr,rv.Input,rv.GlusterEntryDir,rv.Sample,rv.PipeTemp,rv.Force,rv.PipeTempName,rv.Tmp,rv.Env)
    cli.Start()
}
