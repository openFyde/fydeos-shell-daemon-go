package main

import (
	"fmt"
	"os"

	"fydeos.com/shell_daemon/shell_server"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const intro = `
<node>
    <interface name="io.fydeos.ShellInterface">
        <method name="SyncExec">
						<arg direction="in" type="s"/>
            <arg direction="out" type="is"/>
        </method>
				<method name="AsyncExec">
            <arg direction="in" type="s"/>
            <arg direction="out" type="is"/>
        </method>
				<method name="AsyncExec2">
            <arg direction="in" type="s"/>
            <arg direction="out" type="is"/>
        </method>
				<method name="GetTaskState">
            <arg direction="in" type="i"/>
            <arg direction="out" type="is"/>
        </method>
        <method name="GetAsyncTaskOutput">
            <arg direction="in" type="ii"/>
            <arg direction="out" type="is"/>
        </method>
				<method name="GetDaemonState">
            <arg direction="out" type="is"/>
        </method>
				<method name="ForceCloseTask">
            <arg direction="in" type="i"/>
            <arg direction="out" type="is"/>
        </method>
				<method name="EmitNotification">
            <arg direction="in" type="iiis"/>
            <arg direction="out" type="i"/>
        </method>
				<method name="Exit">
        </method>
				<signal name="ShellNotifying">
              <arg type="iiis"/>
        </signal>
    </interface>` + introspect.IntrospectDataString + `</node> `

func main() {
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	server := shell_server.NewServer(conn)
	conn.Export(server, shell_server.DbusPath, shell_server.DbusIface)
	conn.Export(introspect.Introspectable(intro), shell_server.DbusPath,
		"org.freedesktop.DBus.Introspectable")
	reply, err := conn.RequestName("io.fydeos.ShellDaemon", dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}
	fmt.Println("Listening on", shell_server.DbusIface, shell_server.DbusPath, "...")
	select {}
}
