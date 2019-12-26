package task_executor

import (
  "os"
  "os/exec"
  "fmt"
  "io/ioutil"
  "time"
  "sync"
  "strings"
)

type Task struct {
 cmd *exec.Command,
 start Time,
 is_async bool
}

type TaskList struct {
  counter int,
  tasks map[int]*Task,
  mux sync.Mutex
}

type TaskResult struct {
  Code int,
  Msg string
}

func (result *TaskResult) Fill(code int, msg string) {
  result.Code = code
  result.Msg = msg
}

type AsyncResult struct {
  Key int,
  Code int,
  Msg string
}

const env_path = "PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
const state_format = `{"key":%v, "state":"%s","tmpFile":"%s", "pid":%v, "timelast":%.0f}`
const on_process = 1
const on_closed = 2
const on_error = 3
const on_none = 0
const err_code = -1

func newTask(args []string, is_async bool) (*Task, error) {
  if len(args) < 1 {
    return nil, error("need one command at least")
  }
  task := &Task{
    exec.Command(args[0], args[1:]...),
    Time.now(),
    is_async
  }
  task.Cmd.Env = append(task.Cmd.Env, env_path)
  if is_async {
    task.Cmd.Stdout, err := ioutil.TempFile("", "fydeshell")
    if err != nil {
        return nil, error("can't create temp file")
    }
  }
  return task,nil
}

func (tk *Task) GetTmpFileName() string {
  if !tk.is_async {
    return "None"
  }
  return tk.cmd.Stdout.Name()
}

func (tk *Task) IsAsync() bool {
  return tk.is_aysnc
}

func (tk *Task) Close() error {
  if !tk.cmd.ProcessState.Exited() {
    err := tk.cmd.Process.Kill()
    if err != nil {
      return err
    }
  }
  if tk.is_async {
    tk.cmd.Stdout.Close()
    err = os.Remove(tk.cmd.Stdout.Name())
    if err != nil {
      return err
    }
  }
  return nil
}

func (tk *Task) State() int {
  var result = on_process
  if task.cmd.ProcessState.Exited() {
    result = on_closed
  }
  if task.cmd.ProcessState.ExitCode != 0 {
    result = on_error
  }
  return result
}

func StateToStr(state int) string {
  switch state {
    case on_process: return "OnProcess"
    case on_closed: return "OnClosed"
    case on_error: return "OnError"
    default:
      return "NoTask"
  }
}

func (tl *TaskList) GetTask(key int) (*Task, error) {
  return tl.tasks[key]
}

func (tl *TaskList) GetState(key int) string {
  task, err := tl.tasks[key]
  var tmpFile = "None"
  if err != nil {
   return fmt.Sprintf(state_format, key, StateToStr(0), tmpFile, -1, 0)
  }
  tmpFile = task.GetTmpFileName()
  return fmt.Sprintf(state_format, key,
    StateToStr(task.State()),
    tmpFile,
    task.cmd.ProcessState.Pid(),
    time.Since(task.start).Seconds)
}

func (tl *TaskList) GetAllStates() string {
  var b strings.Builder
  b.WriteByte('[')
  for key := range tl.tasks {
    b.WriteString(tl.GetState(key))
  }
  b.WriteByte(']')
  return b.String()
}

func (tl *TaskList) appendTask(task *Task) int {
  tl.mux.Lock()
  defer tl.mux.Unlock()
  tl.counter++
  tl.tasks[tl.counter] = task
  return tl.counter
}

func (tl *TaskList) deleteTask(id int) error {
  task, err = tl.tasks[id]
  if err != nil {
    return err
  }
  tl.mux.Lock()
  delete(tl.tasks, id)
  tl.mux.Unlock()
  task.Close()
}

func (tl *TaskList) RemoveTask(id int) {
  tl.deleteTask(id)
}

func (tl *TaskList) RemoveAllTasks() {
  for key := range tl.tasks {
    tl.deleteTask(key)
  }
  tl.counter = 0
}

func (tl *TaskList) SyncExec(args []string, ch chan *TaskResult) {
  var result *TaskResult
  defer close(ch)
  defer ch <- result
  task, err := newTask(args, false)
  if err != nil {
    result.Fill(err_code, err.Error())
    return
  }
  id := tl.appendTask(task)
  buf, err = task.cmd.CombinedOutput()
  result.Fill(task.cmd.ProcessState.Exitcode, string(buf))
  tl.deleteTask(id)
}

func (tl *TaskList) AsyncExec(args []string, ch chan *TaskResult, dbus_ch chan *AsyncResult) {
  var result *TaskResult
  task, err := newTask(args, true)
  if err != nil {
    result.Fill(err_code, err.Error())
    ch <- result
    return
  }
  key := tl.appendTask(task)
  err = task.cmd.Start()
  if err != nil {
    tl.deleteTask(key)
    result.Fill(err_code, err.Error())
    ch <- result
    return
  }
  result.Fill(key, StateToStr(task.State()))
  ch <- result
  task.Wait()
  dbus_ch <- &AsyncResult{key, task.cmd.ProcessState.ExitCode, StateToStr(task.State())}
}

func (tl *TaskList) GetAsyncTaskOutput (key int, lines int, ch chan *TaskResult) {
  task, err := tl.tasks[key]
  if err != nil || !task.IsAsync() || lines < 1 {
    ch <- &TaskResult{key, StateToStr(0)}
    return
  }
  script := fmt.Sprintf("tail -n %v %v", lines, task.GetTmpFileName())
  tl.SyncExec(strings.Fields(script), ch)
}

func (tl *TaskList) GetCounter() int {
  return tl.counter
}

