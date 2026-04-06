package view

import tea "charm.land/bubbletea/v2"
import input "github.com/kirick13/shellwarden/elements"

type Display interface {
	SetCurrentView(View)
	InnerSize() (int, int)
	SetBWSession(string)
	BWSession() string
}

type View interface {
	Render() string
	OnEnter()
	OnEsc()
	OnKey(string)
	OnMsg(tea.Msg) tea.Cmd
	SetDisplay(Display)
	Inputs() []input.Model
	FocusedInput() int
	SetFocusedInput(int)
}

type BaseView struct {
	Display Display
	inputs  []input.Model
	focused int
}

func (v BaseView) OnEnter() {}
func (v BaseView) OnEsc()   {}
func (v BaseView) OnKey(string) {}
func (v BaseView) OnMsg(tea.Msg) tea.Cmd { return nil }
func (v BaseView) Inputs() []input.Model { return v.inputs }
func (v BaseView) FocusedInput() int     { return v.focused }

func (v *BaseView) SetDisplay(display Display) {
	v.Display = display
}

func (v *BaseView) SetInputs(inputs []input.Model) {
	v.inputs = inputs
}

func (v *BaseView) SetFocusedInput(index int) {
	v.focused = index
}
