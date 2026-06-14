package ui

import (
	"github.com/lxn/walk"
)

// SettingsResult holds the values returned from the settings dialog.
type SettingsResult struct {
	ExePath   string
	ServerURL string
	Accepted  bool
}

// ShowSettingsDialog opens the settings dialog and returns the user's choices.
func ShowSettingsDialog(owner walk.Form, exePath, serverURL string) SettingsResult {
	result := SettingsResult{}

	var dlg *walk.Dialog
	var exeEdit *walk.LineEdit
	var urlEdit *walk.LineEdit

	dlg, _ = walk.NewDialogWithFixedSize(owner)
	dlg.SetTitle("设置")
	dlg.SetSize(walk.Size{Width: 560, Height: 220})

	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 15, VNear: 15, HFar: 15, VFar: 15})
	vbox.SetSpacing(12)
	dlg.SetLayout(vbox)

	// Group box for EXE path
	group, _ := walk.NewGroupBox(dlg)
	group.SetTitle("皎月连配置")
	group.SetLayout(walk.NewVBoxLayout())

	// EXE path row
	exeRow, _ := walk.NewComposite(group)
	exeRowHBoxLayout := walk.NewHBoxLayout()
	exeRowHBoxLayout.SetMargins(walk.Margins{HNear: 6, VNear: 0, HFar: 6, VFar: 0})
	exeRowHBoxLayout.SetSpacing(8)
	exeRow.SetLayout(exeRowHBoxLayout)

	exeLabel, _ := walk.NewLabel(exeRow)
	exeLabel.SetText("EXE路径:")
	exeLabel.SetMinMaxSize(walk.Size{Width: 70, Height: 20}, walk.Size{Width: 70, Height: 20})

	exeEdit, _ = walk.NewLineEdit(exeRow)
	exeEdit.SetText(exePath)
	exeEdit.SetCueBanner("选择皎月连主程序（可选）")

	browseBtn, _ := walk.NewPushButton(exeRow)
	browseBtn.SetText("...")
	browseBtn.SetMinMaxSize(walk.Size{Width: 30, Height: 26}, walk.Size{Width: 30, Height: 26})
	browseBtn.Clicked().Attach(func() {
		dlg2 := new(walk.FileDialog)
		dlg2.Title = "选择服务端程序"
		dlg2.Filter = "EXE文件 (*.exe)|*.exe|所有文件 (*.*)|*.*"
		if ok, _ := dlg2.ShowOpen(owner); ok {
			exeEdit.SetText(dlg2.FilePath)
		}
	})

	// Server URL row
	urlRow, _ := walk.NewComposite(group)
	urlRowHBoxLayout := walk.NewHBoxLayout()
	urlRowHBoxLayout.SetMargins(walk.Margins{HNear: 6, VNear: 4, HFar: 6, VFar: 0})
	urlRowHBoxLayout.SetSpacing(8)
	urlRow.SetLayout(urlRowHBoxLayout)

	urlLabel, _ := walk.NewLabel(urlRow)
	urlLabel.SetText("皎月连WS:")
	urlLabel.SetMinMaxSize(walk.Size{Width: 70, Height: 20}, walk.Size{Width: 70, Height: 20})

	urlEdit, _ = walk.NewLineEdit(urlRow)
	urlEdit.SetText(serverURL)
	urlEdit.SetCueBanner("ws://127.0.0.1:33272/ws")

	// Buttons
	btnComposite, _ := walk.NewComposite(dlg)
	btnLayout := walk.NewHBoxLayout()
	btnLayout.SetSpacing(12)
	btnComposite.SetLayout(btnLayout)

	spacer, _ := walk.NewHSpacer(dlg)
	btnComposite.Children().Add(spacer)

	okBtn, _ := walk.NewPushButton(btnComposite)
	okBtn.SetText("确定")
	okBtn.SetMinMaxSize(walk.Size{Width: 80, Height: 28}, walk.Size{Width: 80, Height: 28})
	okBtn.Clicked().Attach(func() {
		result.ExePath = exeEdit.Text()
		result.ServerURL = urlEdit.Text()
		result.Accepted = true
		dlg.Close(walk.DlgCmdOK)
	})

	cancelBtn, _ := walk.NewPushButton(btnComposite)
	cancelBtn.SetText("取消")
	cancelBtn.SetMinMaxSize(walk.Size{Width: 80, Height: 28}, walk.Size{Width: 80, Height: 28})
	cancelBtn.Clicked().Attach(func() {
		dlg.Close(walk.DlgCmdCancel)
	})

	dlg.Run()
	return result
}
