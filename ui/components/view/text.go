package view

type TextView struct {
	BaseView
	Text string
}

func NewTextView(text string) *TextView {
	return &TextView{
		BaseView: BaseView{},
		Text:     text,
	}
}

func (v *TextView) Render() string {
	return v.Text
}
