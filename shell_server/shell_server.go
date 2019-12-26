package shell_server

import (
  "github.com/godbus/dbus/v5"
  "strings"
  "os"
  "fmt"
  te "./task_executor"
)

var DbusPath = ObjectPath("/io/fydeos/ShellDaemon")
const DbusIface = "io.fydeos.ShellInterface"
const ShellCommand = 1

type DbusServer struct {
  dbus_ch chan *te.AsyncResult,
  excutor *te.TaskList,
  conn *dbus.Conn
}

func NewServer (conn *dbus.Conn) *DbusServer {
  server := &DbusServer{
    make(chan *te.AsyncResult),
    {tasks:  make(map[int]*te.Task)},
    conn
  }
  go func() {
    for {
      select {
        case aResult := <-server.dbus_ch :
          server.ShellNotifying(ShellCommand,
            aResult.key,
            aResult.Code,
            aResult.Msg)
      }
    }
  }
  return &server
}

var ErrCommandNotFound = dbus.NewError("no command script found", nil)

func (server *DbusServer) SyncExec(script string) (int, string, *dbus.Error) {
  args := strings.Fields(script)
  if len(args) < 1 {
    return 0,"",ErrCommandNotFound
  }
  ch := make(chan *te.TaskResult)
  go server.excutor.SyncExec(args, ch)
  result := <-ch
  return result.Code, result.Msg, nil
}

func (server *DbusServer) AsyncExec(script string) (int, string, *dbus.Error) {
  args := strings.Fields(script)
  if len(args) < 1 {
    return 0,"",ErrCommandNotFound
  }
  ch := make(chan *te.TaskResult)
  go server.excutor.AsyncExec(args, ch, server.dbus_ch)
  result := <-ch
  return result.Code, result.Msg, nil
}

func (server *DbusServer) AsyncExec2(script string) (int, string, *dbus.Error) { /*compatible with old script*/
  return AsyncExec(script)
}

func (server *DbusServer) GetTaskState(key int) (int, string, *dbus.Error) {
  task,err := server.excutor.GetTask(key)
  if err != nil {
    return key, te.StateToStr(0), nil
  }
  return key, StateToStr(task.State()), nil
}

func (server *DbusServer) GetAsyncTaskOutput(key int, lines int) (int, string, *dbus.Error) {
  ch := make(chan *te.TaskResult)
  go server.excutor.GetAsyncTaskOutput(key, lines, ch)
  result := <-ch
  return result.Code, result.Msg, nil
}

func (server *DbusServer) GetDaemonState() (int, string, *dbus.Error) {
  return server.excutor.GetCounter(), server.excutor.GetAllStates(), nil
}

func (server *DbusServer) ForceCloseTask(key int) (int, string, *dbus.Error) {
  server.excutor.RemoveTask(key)
  return server.GetTaskState(key)
}

func (server *DbusServer) Exit() {
  server.excutor.RemoveAllTasks()
  os.Exit(1)
}

func (server *DbusServer) ShellNotifying(s_type int, handler int, state int, msg string) error {
  return server.cnn.Emit(DbusPath, DbusIface, s_type, handler, state, msg)
}

func (server *DbusServer) EmitNotification(s_type int, handler int, state int, msg string) (int, *dbus.Error) {
  err := server.ShellNotifying(s_type, handler, state, msg)
  if err != nil {
    return -1, nil
  }
  return 0, nil
}
