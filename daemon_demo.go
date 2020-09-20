package main

import (
        "fmt"
        "time"
        "strings"
        "os"
        "log"
        "daemon"
)
var logFile     string = "daemon_demo.LOG"
var pidFile     string = "daemon_demo.PID"
var daemonName  string = "[daemon_demo EXAMPLE]"
var daemonPwd   *string
var daemonPath  *string

func getDaemonPath() *string {
        myName := strings.Replace(os.Args[0],"./","",-1)
        Path := fmt.Sprintf("%s/%s", *daemonPwd, myName)
        return &Path
}


func getPwd() *string{
        path, err := os.Getwd()
        if err != nil {
            log.Printf("ERROR getPwd: %v\n",err)
        }
        return &path
}


func main() {

        log.Printf("%v BAR:%#v\n", os.Getpid(),os.Getenv("BAR_D3"))
        daemonPwd       = getPwd()
        daemonPath      = getDaemonPath()

        ServiceChild := func (contextPipe *daemon.Context, exit_chan chan int) {
                        i := 0
                        for {
                                if i>30 { break }
                                time.Sleep(2 * time.Second)
                                if (*contextPipe).DebugMode==true {
                                        log.Printf(" %v %v \n",os.Getpid(),i);
                                }
                                i++;
                        }
                        if (*contextPipe).DebugMode==true {
                                log.Printf(" %v done\n",os.Getpid());
                        }
                        exit_chan <- 0
                }

        context := daemon.Context{      DebugMode:      true,
                                        SyslogMode:     true,
                                        LogFile:        logFile,
                                        PidFile:        pidFile,
                                        DaemonName:     daemonName,
                                        DaemonPwd:      *daemonPwd,
                                        DaemonPath:     *daemonPath,
                              }

        var d daemon.Daemon = daemon.Daemon{ Context: &context}
        d.Run(ServiceChild)

}
