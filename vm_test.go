package lua

import (
	"fmt"
	"testing"
)

func TestConcat(t *testing.T) {
	l := NewState()
	BaseOpen(l)
	LoadString(l, "print('hello'..'world')")
	Call(l, 0, 0)
}

func TestProtectedCall(t *testing.T) {
	l := NewState()
	OpenLibraries(l)
	SetHooker(l, func(state *State, ar *Debug) {
		ci := state.callInfo.(*luaCallInfo)
		_ = stack(state.stack[ci.base():state.top])
		_ = ci.code[ci.savedPC].String()
	}, MaskCount, 1)
	LoadString(l, "assert(not pcall(bit32.band, {}))")
	Call(l, 0, 0)
}

func TestLua(t *testing.T) {
	tests := []string{
		"fib",
		"bitwise",
		"math",
		"goto",
		"closure",
		"attrib",
		"strings",
		// "events",
		// "vararg",
	}
	for _, v := range tests {
		t.Log(v)
		l := NewState()
		OpenLibraries(l)
		PushBoolean(l, true)
		PushValue(l, 1)
		PushValue(l, 1)
		SetGlobal(l, "_port")
		SetGlobal(l, "_no32")
		SetGlobal(l, "_noformatA")
		// SetHooker(l, func(state *State, ar *Debug) {
		// 	ci := state.callInfo.(*luaCallInfo)
		// 	p := state.stack[ci.function()].(*luaClosure).prototype
		// 	println(stack(state.stack[ci.base():state.top]))
		// 	println(ci.code[ci.savedPC].String(), p.source, p.lineInfo[ci.savedPC])
		// }, MaskCount, 1)
		LoadFile(l, "fixtures/"+v+".lua", "text")
		if err := ProtectedCall(l, 0, 0, 0); err != nil {
			t.Errorf("'%s' failed: %s", v, err.Error())
		}
	}
}

func TestCanRemoveNilObjectFromStack(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("failed to remove `nil`, %v", r)
		}
	}()

	l := NewState()
	PushString(l, "hello")
	Remove(l, -1)
	PushNil(l)
	Remove(l, -1)
}

func TestTableNext(t *testing.T) {
	l := NewState()
	OpenLibraries(l)
	CreateTable(l, 10, 0)
	for i := 1; i <= 4; i++ {
		PushInteger(l, i)
		PushValue(l, -1)
		SetTable(l, -3)
	}
	if length := LengthEx(l, -1); length != 4 {
		t.Errorf("expected table length to be 4, but was %d", length)
	}
	count := 0
	for PushNil(l); Next(l, -2); count++ {
		if k, v := CheckInteger(l, -2), CheckInteger(l, -1); k != v {
			t.Errorf("key %d != value %d", k, v)
		}
		Pop(l, 1)
	}
	if count != 4 {
		t.Errorf("incorrect iteration count %d in Next()", count)
	}
}

func TestTableUnpack(t *testing.T) {
	l := NewState()
	OpenLibraries(l)
	LoadString(l, "local x, y = table.unpack({-10,0}); assert(x == -10 and y == 0)")
	Call(l, 0, 0)
}

func TestBase(t *testing.T) {
	s := `
    print("hello world\n")
    assert(true)`
	l := NewState()
	BaseOpen(l)
	LoadString(l, s)
	Call(l, 0, 0)
}

func TestError(t *testing.T) {
	l := NewState()
	BaseOpen(l)
	errorHandled := false
	program := "error('error')"
	PushGoFunction(l, func(l *State) int {
		if Top(l) == 0 {
			t.Error("error handler received no arguments")
		} else if errorMessage, ok := ToString(l, -1); !ok {
			t.Errorf("error handler received %s instead of string", TypeNameOf(l, -1))
		} else if errorMessage != program+":1: error" {
			t.Errorf("error handler received '%s' instead of 'error'", errorMessage)
		}
		errorHandled = true
		return 1
	})
	LoadString(l, program)
	ProtectedCall(l, 0, 0, -2)
	if !errorHandled {
		t.Error("error not handled")
	}
}

func Example() {
	type step struct {
		name     string
		function interface{}
	}
	steps := []step{}
	l := NewState()
	BaseOpen(l)
	_ = NewMetaTable(l, "stepMetaTable")
	SetFunctions(l, []RegistryFunction{{"__newindex", func(l *State) int {
		k, v := CheckString(l, 2), ToValue(l, 3)
		steps = append(steps, step{name: k, function: v})
		return 0
	}}}, 0)
	PushUserData(l, steps)
	PushValue(l, -1)
	SetGlobal(l, "step")
	SetMetaTableNamed(l, "stepMetaTable")
	LoadString(l, `step.request_tracking_js = function ()
	  get(config.domain..'/javascripts/shopify_stats.js')
	end`)
	Call(l, 0, 0)
	fmt.Println(steps[0].name)
	// Output: request_tracking_js
}
