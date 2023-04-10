package task_executor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

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

type Task struct {
	cmd      *exec.Cmd
	start    time.Time
	is_async bool
	tmpFile  string
}

type TaskList struct {
	counter int
	tasks   map[int]*Task
	mux     *sync.Mutex
}

type TaskResult struct {
	Code int
	Msg  string
}

func (result *TaskResult) Fill(code int, msg string) {
	result.Code = code
	result.Msg = strings.Map(func(r rune) rune {
		if r != 0 {
			return r
		}
		return -1
	}, msg)
}

type AsyncResult struct {
	Key  int
	Code int
	Msg  string
}

const env_path = "PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
const state_format = `{"key":%v, "state":"%s","tmpFile":"%s", "pid":%v, "timelast":%.0f}`
const on_process = 1
const on_closed = 2
const on_error = 3
const on_none = 0
const err_code = -1

func (tk *Task) ExitCode() int {
	if tk.cmd.ProcessState == nil {
		return err_code
	}
	if tk.cmd.ProcessState.Success() {
		return 0
	}
	return err_code
}

func NewTaskList() *TaskList {
	return &TaskList{0,
		make(map[int]*Task),
		&sync.Mutex{},
	}
}

func newTask(args []string, is_async bool) (*Task, error) {
	if len(args) < 1 {
		return nil, errors.New("need one command at least")
	}
	task := &Task{
		exec.Command(args[0], args[1:]...),
		time.Now(),
		is_async,
		"None",
	}
	task.cmd.Env = append(task.cmd.Env, env_path)
	if is_async {
		tempFile, err := ioutil.TempFile("", "fydeshell")
		task.cmd.Stdout = tempFile
		task.cmd.Stderr = tempFile
		task.tmpFile = tempFile.Name()
		if err != nil {
			return nil, errors.New("can't create temp file")
		}
	}
	return task, nil
}

func (tk *Task) GetTmpFileName() string {
	return tk.tmpFile
}

func (tk *Task) IsAsync() bool {
	return tk.is_async
}

func (tk *Task) Close() error {
	if (!tk.is_async && tk.cmd.ProcessState != nil && !tk.cmd.ProcessState.Exited()) || (tk.is_async && tk.cmd.Process != nil) {
		err := tk.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}
	if tk.is_async {
		err := os.Remove(tk.tmpFile)
		dPrintln("remove", tk.tmpFile, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tk *Task) State() int {
	var result = on_process
	if tk.cmd.ProcessState != nil && tk.cmd.ProcessState.Exited() {
		result = on_closed
		if !tk.cmd.ProcessState.Success() {
			result = on_error
		}
	}
	return result
}

func StateToStr(state int) string {
	switch state {
	case on_process:
		return "OnProcess"
	case on_closed:
		return "OnClosed"
	case on_error:
		return "OnError"
	default:
		return "NoTask"
	}
}

func (tl *TaskList) GetTask(key int) (*Task, error) {
	task, ok := tl.tasks[key]
	if !ok {
		return nil, errors.New("NoTask")
	}
	return task, nil
}

func (tl *TaskList) GetState(key int) string {
	task, ok := tl.tasks[key]
	var tmpFile = "None"
	if !ok {
		return fmt.Sprintf(state_format, key, StateToStr(0), tmpFile, -1, 0.0)
	}
	tmpFile = task.GetTmpFileName()
	return fmt.Sprintf(state_format, key,
		StateToStr(task.State()),
		tmpFile,
		task.cmd.Process.Pid,
		time.Since(task.start).Seconds())
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
	task, ok := tl.tasks[id]
	if !ok {
		return errors.New("NoTask")
	}
	tl.mux.Lock()
	delete(tl.tasks, id)
	tl.mux.Unlock()
	task.Close()
	return nil
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
	result := &TaskResult{}
	defer func() {
		ch <- result
		close(ch)
	}()
	task, err := newTask(args, false)
	if err != nil {
		result.Fill(err_code, err.Error())
		return
	}
	id := tl.appendTask(task)
	buf, err := task.cmd.CombinedOutput()
	if err == nil {
		result.Fill(task.ExitCode(), string(buf))
	} else {
		result.Fill(task.ExitCode(), err.Error())
	}
	tl.deleteTask(id)
	dPrintln(trace(), result)
}

func (tl *TaskList) AsyncExec(args []string, ch chan *TaskResult, dbus_ch chan *AsyncResult) {
	result := &TaskResult{}
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
	task.cmd.Wait()
	dPrintln(trace(), key, task.ExitCode(), StateToStr(task.State()))
	dbus_ch <- &AsyncResult{key, task.State(), StateToStr(task.State())}
}

func (tl *TaskList) GetAsyncTaskOutput(key int, lines int, ch chan *TaskResult) {
	result := &TaskResult{}
	defer func() {
		ch <- result
		close(ch)
	}()
	task, ok := tl.tasks[key]
	if !ok || !task.IsAsync() || lines < 1 {
		result.Fill(on_none, StateToStr(on_none))
		return
	}
	script := fmt.Sprintf("tail -n %v %v", lines, task.GetTmpFileName())
	taskTmp, err := newTask(strings.Fields(script), false)
	if err != nil {
		result.Fill(err_code, err.Error())
		return
	}
	buf, err := taskTmp.cmd.CombinedOutput()
	if err != nil {
		result.Fill(err_code, err.Error())
		return
	}
	result.Fill(task.State(), string(buf))
	if task.State() == on_closed || task.State() == on_error {
		tl.RemoveTask(key)
	}
}

func (tl *TaskList) GetCounter() int {
	return tl.counter
}
