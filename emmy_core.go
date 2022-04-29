package lua_debugger

import (
	"io"
	"log"
	"sync/atomic"

	lua "github.com/yuin/gopher-lua"
)

// openConn tracks open connection
// This is necessary because otherwise debugger
// (at least in InteliJ) will not work with multiple connected debuggers
// at the same time. So we try to guard ourselves
// to not deadlock incoming requests to NNBB Hub.
var openConn int32

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

const (
	KeyDebuggerFcd = "__Debugger_Fcd"
)

func (f *Facade) Connect(L *lua.LState) int {
	host := L.CheckString(1)
	port := L.CheckNumber(2)
	// If no connections opened yet - we can open one.
	if atomic.CompareAndSwapInt32(&openConn, 0, 1) {
		if err := f.TcpConnect(L, host, int(port)); err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
	}

	L.Push(lua.LTrue)
	return 1
}

func (f *Facade) Loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, map[string]lua.LGFunction{
		"tcpConnect": f.Connect,
	})
	L.Push(t)
	return 1
}

func Preload(L *lua.LState) io.Closer {
	// Creating facade here to track what to close
	// when debugger should be finished.
	fcd := newFacade()
	fcdUd := L.NewUserData()
	fcdUd.Value = fcd
	L.SetField(L.Get(lua.RegistryIndex), KeyDebuggerFcd, fcdUd)

	L.PreloadModule("emmy_core", fcd.Loader)

	return fcd
}
