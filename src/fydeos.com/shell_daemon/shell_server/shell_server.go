package shell_server

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	te "fydeos.com/shell_daemon/shell_server/task_executor"
	"github.com/godbus/dbus/v5"
)

var DbusPath = dbus.ObjectPath("/io/fydeos/ShellDaemon")

const DbusIface = "io.fydeos.ShellInterface"
const ShellCommand = 1

// Debug related begin
const debug = false

func trace() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	return fn.Name()
}

func dPrintln(a ...interface{}) {
	if debug {
		fmt.Println(a...)
	}
}

//Debug related end

type DbusServer struct {
	dbus_ch chan *te.AsyncResult
	excutor *te.TaskList
	conn    *dbus.Conn
}

func NewServer(conn *dbus.Conn) *DbusServer {
	server := &DbusServer{
		make(chan *te.AsyncResult),
		te.NewTaskList(),
		conn,
	}
	go server.ListenAsyncCh()
	return server
}

func (server *DbusServer) ListenAsyncCh() {
	for {
		select {
		case aResult := <-server.dbus_ch:
			dPrintln(trace(), aResult)
			server.ShellNotifying(ShellCommand,
				aResult.Key,
				aResult.Code,
				aResult.Msg)
		}
	}
}

var ErrCommandNotFound = dbus.NewError("no command script found", nil)
var EmptyResult = &te.TaskResult{0, ""}

func (server *DbusServer) SyncExec(script string) (*te.TaskResult, *dbus.Error) {
	dPrintln(trace(), script)
	args := strings.Fields(script)
	if len(args) < 1 {
		return EmptyResult, ErrCommandNotFound
	}
	ch := make(chan *te.TaskResult)
	go server.excutor.SyncExec(args, ch)
	result := <-ch
	dPrintln(trace(), result)
	return result, nil
}

func (server *DbusServer) AsyncExec(script string) (*te.TaskResult, *dbus.Error) {
	args := strings.Fields(script)
	if len(args) < 1 {
		return EmptyResult, ErrCommandNotFound
	}
	ch := make(chan *te.TaskResult)
	go server.excutor.AsyncExec(args, ch, server.dbus_ch)
	result := <-ch
	return result, nil
}

func (server *DbusServer) AsyncExec2(script string) (*te.TaskResult, *dbus.Error) { /*compatible with old script*/
	return server.AsyncExec(script)
}

func (server *DbusServer) GetTaskState(key int) (*te.TaskResult, *dbus.Error) {
	_, err := server.excutor.GetTask(key)
	if err != nil {
		return EmptyResult, nil
	}
	return &te.TaskResult{key, server.excutor.GetState(key)}, nil
}

func (server *DbusServer) GetAsyncTaskOutput(key int, lines int) (*te.TaskResult, *dbus.Error) {
	ch := make(chan *te.TaskResult)
	go server.excutor.GetAsyncTaskOutput(key, lines, ch)
	result := <-ch
	dPrintln(trace(), result)
	return result, nil
}

func (server *DbusServer) GetDaemonState() (*te.TaskResult, *dbus.Error) {
	return &te.TaskResult{server.excutor.GetCounter(), server.excutor.GetAllStates()}, nil
}

func (server *DbusServer) ForceCloseTask(key int) (*te.TaskResult, *dbus.Error) {
	server.excutor.RemoveTask(key)
	return server.GetTaskState(key)
}

func (server *DbusServer) Exit() {
	server.excutor.RemoveAllTasks()
	os.Exit(1)
}

/*
type FydeNotification struct {
  s_type int
  handler int
  state int
  msg string
}
*/
func (server *DbusServer) ShellNotifying(s_type int, handler int, state int, msg string) error {
	dPrintln(trace(), s_type, handler, state, msg)
	return server.conn.Emit(DbusPath, DbusIface+".ShellNotifying", s_type, handler, state, msg)
}

func (server *DbusServer) EmitNotification(s_type int, handler int, state int, msg string) (int, *dbus.Error) {
	err := server.ShellNotifying(s_type, handler, state, msg)
	if err != nil {
		return -1, nil
	}
	return 0, nil
}
