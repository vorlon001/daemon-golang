package daemon

import (
        "fmt"
        "io"
        "syscall"
        "os/signal"
        "encoding/json"
        "os"
        "log"
        "log/syslog"
        "bytes"
        "os/exec"
)

type Context struct {
        DebugMode       bool
        SyslogMode      bool
        LogFile         string
        PidFile         string
        DaemonName      string
        DaemonPwd       string
        DaemonPath      string
        filePid         *os.File
}

type Daemon struct {
        Context *Context
}

func (d *Daemon) initSignals ( exit_chan chan int) {
        signal_chan := make(chan os.Signal, 1)

        signal.Notify(signal_chan,
                syscall.SIGHUP,
                syscall.SIGINT,
                syscall.SIGTERM,
                syscall.SIGQUIT)
        if (*d.Context).DebugMode==true {
                log.Printf("%#v\n",os.Getpid())
        }
        go func() {
                for {
                        s := <-signal_chan
                        switch s {
                        // kill -SIGHUP XXXX
                        case syscall.SIGHUP:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGHUP %v\n",os.Getpid())
                                }
                        // kill -SIGINT XXXX or Ctrl+c
                        case syscall.SIGINT:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGINT %v \n",os.Getpid())
                                }
                                exit_chan <- 0
                        // kill -SIGTERM XXXX
                        case syscall.SIGTERM:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGTERM force stop %v \n",os.Getpid())
                                }
                                exit_chan <- 0

                        // kill -SIGQUIT XXXX
                        case syscall.SIGQUIT:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGQUIT stop and core dump %v \n",os.Getpid())
                                }
                                exit_chan <- 0

                        default:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("Unknown signal.",os.Getpid())
                                }
                                exit_chan <- 1
                        }
                }
        }()

}


func (d *Daemon) initIoMux(ioW *[]io.Writer) *io.Writer {
        multi := io.MultiWriter( *ioW... )
        return &multi
}

func (d *Daemon) initLogFile( childMode bool) {

        ioW := []io.Writer{ os.Stdout }

        if childMode==false {
                logFile, err := os.OpenFile((*d.Context).LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640));
                if err!=nil {
                        log.Printf("%v os.OpenFile(logFile) ERROR:%v\n",os.Getpid(), err )
                }
                ioW = append( ioW, logFile )
        }

        if (*d.Context).SyslogMode==true {
                logSyslog, err := syslog.New(syslog.LOG_NOTICE, (*d.Context).DaemonName)
                if err!=nil {
                        log.Printf("%v os.OpenFile(logFile) ERROR:%v\n",os.Getpid(), err )
                }
                ioW = append( ioW, logSyslog )
        }

        multi := d.initIoMux(&ioW)

        log.SetOutput(*multi)

        if (*d.Context).DebugMode==true {
                log.SetFlags(log.LstdFlags | log.Lshortfile)
        }
}


func (d *Daemon) startDaemon(type_start bool, args ...string) (p *os.Process, err error) {

        Dir := (*d.Context).DaemonPwd

        if type_start==true {
                os.Setenv("FOO", "1")
                if (*d.Context).DebugMode==true {
                        log.Printf("%v FOO:%#v\n", os.Getpid(),os.Getenv("FOO"))
                        log.Printf("%v BAR:%#v\n", os.Getpid(),os.Getenv("BAR"))
                }
        }
        Env := os.Environ()

        if (*d.Context).DebugMode==true {
                log.Printf("%v Dir:%#v\n", os.Getpid(), Dir );
                log.Printf("%v Env:%#v\n", os.Getpid(), Env );
        }

        nullFile,_ := os.Open(os.DevNull);
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.Open(os.DevNull) is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 142. os.Open(os.DevNull) ERROR:%v \n", os.Getpid(), err)
        }

        rpipe, wpipe, err := os.Pipe();
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.Pipe is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 143. os.Pipe ERROR:%v \n", os.Getpid(), err)
        }
        logFileDaemon, err := os.OpenFile( (*d.Context).LogFile , os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640));
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.OpenFile(logFile) is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 144. os.OpenFile(logFile) ERROR:%v \n", os.Getpid(), err)
        }

        if args[0], err = exec.LookPath(args[0]); err == nil {
                var procAttr os.ProcAttr = os.ProcAttr{ Dir: Dir,
                                                        Env: Env,
                                                        Files: []*os.File{
                                                                                rpipe,                  // (0) stdin
                                                                                logFileDaemon,          // (1) stdout
                                                                                logFileDaemon,          // (2) stderr
                                                                                nullFile,               // (3) dup on fd 0 after initialization
                                                                        },
                                                        }
                if p, err := os.StartProcess(args[0], []string{"[go-daemon sample]"}/*args*/, &procAttr);err == nil {

                        if (*d.Context).DebugMode==true {
                                log.Printf("%v SEND context:%#v \n", os.Getpid(), (*d.Context) )
                        }
                        bufJson := new(bytes.Buffer)
                        if err = json.NewEncoder(bufJson ).Encode(*d.Context); err==nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v json.NewEncoder..Encode is DONE \n", os.Getpid())
                                }
                        } else {
                                log.Printf("%v POINT 153. json.NewEncoder..Encode ERROR:%v \n", os.Getpid(), err)
                        }

                        log.Printf("%v SEND JSON %#v \n", os.Getpid(), string(bufJson.Bytes()) )
                        if sendDone, err := fmt.Fprint(wpipe, string(bufJson.Bytes()) );err == nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v SEND JSON to DAEMON:%v \n", os.Getpid(), sendDone)
                                }
                        } else {
                                log.Printf("%v POINT 151. ERROR SEND JSON to DAEMON:%v \n", os.Getpid(), err)
                        }

                        if sendDone, err := fmt.Fprint(wpipe, "\n\n" );err == nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v SEND JSON to DAEMON:%v \n", os.Getpid(), sendDone)
                                }
                        } else {
                                log.Printf("%v POINT 152. ERROR SEND JSON to DAEMON:%v \n", os.Getpid(), err)
                        }
                        if err = wpipe.Close(); err==nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v wpipe.Close is DONE \n", os.Getpid())
                                }
                        } else {
                                log.Printf("%v POINT 153. wpipe.Close ERROR:%v \n", os.Getpid(), err)
                        }
                        return p, nil
                } else {
                        log.Printf("%v ERROR RUN DAEMON:%v \n", os.Getpid(), err)
                }
        }

        return nil, err
}

func (d *Daemon) runStart() {
        os.Setenv("BAR_D3", "1")
        d.initLogFile(false);
        if proc, err := d.startDaemon(true, (*d.Context).DaemonPath ); err == nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v %#v\n",  os.Getpid(),    proc    );
                        log.Printf("%v PID CHILD %v\n", os.Getpid(), proc.Pid   )
                }
        }
}


func (d *Daemon) shutdownDaemon() error {
        err := syscall.Flock( int((*d.Context).filePid.Fd()), syscall.LOCK_UN|syscall.LOCK_NB)
        if err != nil {
                log.Printf("%v %v\n",os.Getpid(),err)
                return err
        }
        e := os.Remove( (*d.Context).PidFile )
        if e != nil {
                log.Printf("%v %v\n",os.Getpid(),e)
                return err
        }
        return nil
}

func  (d *Daemon) initChildContext() *Context{
        log.SetFlags(log.LstdFlags | log.Lshortfile)
        log.Printf("%v ARG START ENV:%#v\n", os.Getpid(),os.Getenv("BAR_D3") )
        d.Context = &Context{}

        decoder := json.NewDecoder(os.Stdin)
        if err := decoder.Decode(d.Context); err != nil {
                log.Printf("%v ERROR DECODE CONTEXT:%v\n",os.Getpid(),err)
                return nil
        } else {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v Context:%#v \n",os.Getpid(), (*d.Context))
                }
        }

        d.initLogFile(true);
        if (*d.Context).DebugMode==true {
                log.Printf("%v contextPipe.PidFile:%v\n",os.Getpid(), (*d.Context).PidFile);
        }
        if len((*d.Context).PidFile) > 0 {
                var err error
                (*d.Context).filePid, err = os.OpenFile( (*d.Context).PidFile , os.O_WRONLY|os.O_CREATE, os.FileMode(0640));
                if err != nil {
                        log.Printf("%v POINT 9. FAIL  os.OpenFile( contextPipe.PidFile) ERROR:%v \n",os.Getpid(), err)
                        return nil
                }
                err = syscall.Flock( int((*d.Context).filePid.Fd()) , syscall.LOCK_EX|syscall.LOCK_NB)
                if err != nil && err.Error() == "resource temporarily unavailable" {
                        log.Printf("%v POINT 10. FAIL INIT. PROCESS IS RUN. ERROR:%v\n",os.Getpid(),err)
                        log.Printf("%v %v\n",os.Getpid(),err)
                        return nil
                }
                if _, err := fmt.Fprintln((*d.Context).filePid, os.Getpid()); err != nil {
                        log.Printf("%v POINT 11. FAIL INIT. PROCESS IS RUN. ERROR:%v\n",os.Getpid(),err)
                        log.Printf("%v %v\n",os.Getpid(),err)
                        return nil
                }
        } else {
                log.Printf("%v POINT 12. FAIL INIT. PROCESS IS RUN.\n",os.Getpid())
                return nil
        }
        return d.Context
}


func (d *Daemon) childWait(exit_chan chan int) {
        code := <-exit_chan
        err := d.shutdownDaemon();
        if err!=nil {
                code = 1
        }
        os.Exit(code)
}

func (d *Daemon) Run(Service func (*Context, chan int)) {
        if len(os.Getenv("BAR_D3"))>0 {
                d.Context = d.initChildContext()
                if d.Context==nil {
                        return
                }

                exit_chan := make(chan int)
                go d.initSignals( exit_chan );

                go Service(d.Context,exit_chan)

                d.childWait(exit_chan)
        } else {
                d.runStart()
        }
}



vorlon@backup-script-server:~/git/golang/GO/EXAMPLE7/daemon_demo_8/src/daemon$ cat daemon.go
package daemon

import (
        "fmt"
        "io"
        "syscall"
        "os/signal"
        "encoding/json"
        "os"
        "log"
        "log/syslog"
        "bytes"
        "os/exec"
)

type Context struct {
        DebugMode       bool
        SyslogMode      bool
        LogFile         string
        PidFile         string
        DaemonName      string
        DaemonPwd       string
        DaemonPath      string
        filePid         *os.File
}

type Daemon struct {
        Context *Context
}

func (d *Daemon) initSignals ( exit_chan chan int) {
        signal_chan := make(chan os.Signal, 1)

        signal.Notify(signal_chan,
                syscall.SIGHUP,
                syscall.SIGINT,
                syscall.SIGTERM,
                syscall.SIGQUIT)
        log.Printf("%#v\n",os.Getpid())
        go func() {
                for {
                        s := <-signal_chan
                        switch s {
                        // kill -SIGHUP XXXX
                        case syscall.SIGHUP:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGHUP %v\n",os.Getpid())
                                }
                        // kill -SIGINT XXXX or Ctrl+c
                        case syscall.SIGINT:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGINT %v \n",os.Getpid())
                                }
                                exit_chan <- 0
                        // kill -SIGTERM XXXX
                        case syscall.SIGTERM:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGTERM force stop %v \n",os.Getpid())
                                }
                                exit_chan <- 0

                        // kill -SIGQUIT XXXX
                        case syscall.SIGQUIT:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("syscall.SIGQUIT stop and core dump %v \n",os.Getpid())
                                }
                                exit_chan <- 0

                        default:
                                if (*d.Context).DebugMode==true {
                                        log.Printf("Unknown signal.",os.Getpid())
                                }
                                exit_chan <- 1
                        }
                }
        }()

}


func (d *Daemon) initIoMux(ioW *[]io.Writer) *io.Writer {
        multi := io.MultiWriter( *ioW... )
        return &multi
}

func (d *Daemon) initLogFile( childMode bool) {

        ioW := []io.Writer{ os.Stdout }

        if childMode==false {
                logFile, err := os.OpenFile((*d.Context).LogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640));
                if err!=nil {
                        log.Printf("%v os.OpenFile(logFile) ERROR:%v\n",os.Getpid(), err )
                }
                ioW = append( ioW, logFile )
        }

        if (*d.Context).SyslogMode==true {
                logSyslog, err := syslog.New(syslog.LOG_NOTICE, (*d.Context).DaemonName)
                if err!=nil {
                        log.Printf("%v os.OpenFile(logFile) ERROR:%v\n",os.Getpid(), err )
                }
                ioW = append( ioW, logSyslog )
        }

        multi := d.initIoMux(&ioW)

        log.SetOutput(*multi)

        if (*d.Context).DebugMode==true {
                log.SetFlags(log.LstdFlags | log.Lshortfile)
        }
}


func (d *Daemon) startDaemon(type_start bool, args ...string) (p *os.Process, err error) {

        Dir := (*d.Context).DaemonPwd

        if type_start==true {
                os.Setenv("FOO", "1")
                if (*d.Context).DebugMode==true {
                        log.Printf("%v FOO:%#v\n", os.Getpid(),os.Getenv("FOO"))
                        log.Printf("%v BAR:%#v\n", os.Getpid(),os.Getenv("BAR"))
                }
        }
        Env := os.Environ()

        if (*d.Context).DebugMode==true {
                log.Printf("%v Dir:%#v\n", os.Getpid(), Dir );
                log.Printf("%v Env:%#v\n", os.Getpid(), Env );
        }

        nullFile,_ := os.Open(os.DevNull);
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.Open(os.DevNull) is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 142. os.Open(os.DevNull) ERROR:%v \n", os.Getpid(), err)
        }

        rpipe, wpipe, err := os.Pipe();
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.Pipe is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 143. os.Pipe ERROR:%v \n", os.Getpid(), err)
        }
        logFileDaemon, err := os.OpenFile( (*d.Context).LogFile , os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640));
        if err==nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v os.OpenFile(logFile) is DONE\n", os.Getpid() )
                }
        } else {
                log.Printf("%v POINT 144. os.OpenFile(logFile) ERROR:%v \n", os.Getpid(), err)
        }

        if args[0], err = exec.LookPath(args[0]); err == nil {
                var procAttr os.ProcAttr = os.ProcAttr{ Dir: Dir,
                                                        Env: Env,
                                                        Files: []*os.File{
                                                                                rpipe,                  // (0) stdin
                                                                                logFileDaemon,          // (1) stdout
                                                                                logFileDaemon,          // (2) stderr
                                                                                nullFile,               // (3) dup on fd 0 after initialization
                                                                        },
                                                        }
                if p, err := os.StartProcess(args[0], []string{"[go-daemon sample]"}/*args*/, &procAttr);err == nil {

                        if (*d.Context).DebugMode==true {
                                log.Printf("%v SEND context:%#v \n", os.Getpid(), (*d.Context) )
                        }
                        bufJson := new(bytes.Buffer)
                        if err = json.NewEncoder(bufJson ).Encode(*d.Context); err==nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v json.NewEncoder..Encode is DONE \n", os.Getpid())
                                }
                        } else {
                                log.Printf("%v POINT 153. json.NewEncoder..Encode ERROR:%v \n", os.Getpid(), err)
                        }

                        log.Printf("%v SEND JSON %#v \n", os.Getpid(), string(bufJson.Bytes()) )
                        if sendDone, err := fmt.Fprint(wpipe, string(bufJson.Bytes()) );err == nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v SEND JSON to DAEMON:%v \n", os.Getpid(), sendDone)
                                }
                        } else {
                                log.Printf("%v POINT 151. ERROR SEND JSON to DAEMON:%v \n", os.Getpid(), err)
                        }

                        if sendDone, err := fmt.Fprint(wpipe, "\n\n" );err == nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v SEND JSON to DAEMON:%v \n", os.Getpid(), sendDone)
                                }
                        } else {
                                log.Printf("%v POINT 152. ERROR SEND JSON to DAEMON:%v \n", os.Getpid(), err)
                        }
                        if err = wpipe.Close(); err==nil {
                                if (*d.Context).DebugMode==true {
                                        log.Printf("%v wpipe.Close is DONE \n", os.Getpid())
                                }
                        } else {
                                log.Printf("%v POINT 153. wpipe.Close ERROR:%v \n", os.Getpid(), err)
                        }
                        return p, nil
                } else {
                        log.Printf("%v ERROR RUN DAEMON:%v \n", os.Getpid(), err)
                }
        }

        return nil, err
}

func (d *Daemon) runStart() {
        os.Setenv("BAR_D3", "1")
        d.initLogFile(false);
        if proc, err := d.startDaemon(true, (*d.Context).DaemonPath ); err == nil {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v %#v\n",  os.Getpid(),    proc    );
                        log.Printf("%v PID CHILD %v\n", os.Getpid(), proc.Pid   )
                }
        }
}


func (d *Daemon) shutdownDaemon() error {
        err := syscall.Flock( int((*d.Context).filePid.Fd()), syscall.LOCK_UN|syscall.LOCK_NB)
        if err != nil {
                log.Printf("%v %v\n",os.Getpid(),err)
                return err
        }
        e := os.Remove( (*d.Context).PidFile )
        if e != nil {
                log.Printf("%v %v\n",os.Getpid(),e)
                return err
        }
        return nil
}

func  (d *Daemon) initChildContext() *Context{
        log.SetFlags(log.LstdFlags | log.Lshortfile)
        log.Printf("%v ARG START ENV:%#v\n", os.Getpid(),os.Getenv("BAR_D3") )
        d.Context = &Context{}

        decoder := json.NewDecoder(os.Stdin)
        if err := decoder.Decode(d.Context); err != nil {
                log.Printf("%v ERROR DECODE CONTEXT:%v\n",os.Getpid(),err)
                return nil
        } else {
                if (*d.Context).DebugMode==true {
                        log.Printf("%v Context:%#v \n",os.Getpid(), (*d.Context))
                }
        }

        d.initLogFile(true);
        if (*d.Context).DebugMode==true {
                log.Printf("%v contextPipe.PidFile:%v\n",os.Getpid(), (*d.Context).PidFile);
        }
        if len((*d.Context).PidFile) > 0 {
                var err error
                (*d.Context).filePid, err = os.OpenFile( (*d.Context).PidFile , os.O_WRONLY|os.O_CREATE, os.FileMode(0640));
                if err != nil {
                        log.Printf("%v POINT 9. FAIL  os.OpenFile( contextPipe.PidFile) ERROR:%v \n",os.Getpid(), err)
                        return nil
                }
                err = syscall.Flock( int((*d.Context).filePid.Fd()) , syscall.LOCK_EX|syscall.LOCK_NB)
                if err != nil && err.Error() == "resource temporarily unavailable" {
                        log.Printf("%v POINT 10. FAIL INIT. PROCESS IS RUN. ERROR:%v\n",os.Getpid(),err)
                        log.Printf("%v %v\n",os.Getpid(),err)
                        return nil
                }
                if _, err := fmt.Fprintln((*d.Context).filePid, os.Getpid()); err != nil {
                        log.Printf("%v POINT 11. FAIL INIT. PROCESS IS RUN. ERROR:%v\n",os.Getpid(),err)
                        log.Printf("%v %v\n",os.Getpid(),err)
                        return nil
                }
        } else {
                log.Printf("%v POINT 12. FAIL INIT. PROCESS IS RUN.\n",os.Getpid())
                return nil
        }
        return d.Context
}


func (d *Daemon) childWait(exit_chan chan int) {
        code := <-exit_chan
        err := d.shutdownDaemon();
        if err!=nil {
                code = 1
        }
        os.Exit(code)
}

func (d *Daemon) Run(Service func (*Context, chan int)) {
        if len(os.Getenv("BAR_D3"))>0 {
                d.Context = d.initChildContext()
                if d.Context==nil {
                        return
                }

                exit_chan := make(chan int)
                go d.initSignals( exit_chan );

                go Service(d.Context,exit_chan)

                d.childWait(exit_chan)
        } else {
                d.runStart()
        }
}
