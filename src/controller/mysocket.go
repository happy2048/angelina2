package controller 
import(
	"net"
	"myutils"
)
func (ctrl *Controller) ListenSocketService() {
 	service := ctrl.Service
	udpAddr,err := net.ResolveUDPAddr("udp4",service)
	if err != nil {
		myutils.Print("Error","create socket server failed,reason: " + err.Error(),true)
	}
	conn,err := net.ListenUDP("udp",udpAddr)
	if err != nil {
		myutils.Print("Error","create socket server failed,reason: " + err.Error(),true)
	}
	for{
		ctrl.HandleRunnerMessage(conn)
	}
}
func (ctrl *Controller) HandleRunnerMessage(conn *net.UDPConn) {
	var buf [512]byte
	n,addr,err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		myutils.Print("Error","read udp data failed,reason: " + err.Error(),false)
		return 
	}
	go ctrl.PushMessage(string(buf[0:n]))
	conn.WriteToUDP([]byte("received"),addr)
}
