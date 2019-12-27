# What's shell daemon for?

 It is a dbus server provides the ablility to execute shell command and return
 the result to Chrome.

## Dbus information:

Bus Type: system
Bus Destination: io.fydeos.ShellDaemon
Bus Path: `/io/fydeos/ShellDaemon`
Bus Interface: io.fydeos.ShellInterface

## Dbus motheds:

1. `result SyncExec(cmd)`
   cmd: String (shell command)
   result: Tuple (code , stdout) code: integer (shell return code)
2. `state AsyncExec(cmd)`
   cmd: as 1
   state: Tuple (taskid, state) taskid: integer state: String
3. `state GetTaskState(taskid)` checkout if the async-task is done.
4. `result GetTaskOutput(taskid, lines)` get async-task's output
5. `state ForceCloseTask(taskid)` force close task
6. `state GetDaemonState()` get all running tasks information
7. `state Exit()` close all tasks and finished server.
