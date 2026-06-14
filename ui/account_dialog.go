package ui

import (
	"github.com/lxn/walk"
	"GoWork/config"
)

// AccountResult holds the values returned from the account dialog.
type AccountResult struct {
	Account  config.Account
	Accepted bool
}

// ShowAccountDialog opens the add/edit account dialog.
func ShowAccountDialog(owner walk.Form, isEdit bool, acc config.Account) AccountResult {
	result := AccountResult{}

	var dlg *walk.Dialog
	var userEdit *walk.LineEdit
	var pwdEdit *walk.LineEdit

	title := "添加账户"
	if isEdit {
		title = "编辑账户"
	}

	dlg, _ = walk.NewDialogWithFixedSize(owner)
	dlg.SetTitle(title)

	vbox := walk.NewVBoxLayout()
	vbox.SetMargins(walk.Margins{HNear: 12, VNear: 12, HFar: 12, VFar: 12})
	vbox.SetSpacing(8)
	dlg.SetLayout(vbox)

	// Form fields
	userLabel, _ := walk.NewLabel(dlg)
	userLabel.SetText("用户名:")
	userEdit, _ = walk.NewLineEdit(dlg)
	userEdit.SetText(acc.Username)

	pwdLabel, _ := walk.NewLabel(dlg)
	pwdLabel.SetText("密码:")
	pwdEdit, _ = walk.NewLineEdit(dlg)
	pwdEdit.SetText(acc.Password)
	pwdEdit.SetPasswordMode(true)

	// Buttons
	btnComposite, _ := walk.NewComposite(dlg)
	btnLayout := walk.NewHBoxLayout()
	btnComposite.SetLayout(btnLayout)

	spacer, _ := walk.NewHSpacer(dlg)
	btnComposite.Children().Add(spacer)

	okBtn, _ := walk.NewPushButton(btnComposite)
	okBtn.SetText("确定")
	okBtn.Clicked().Attach(func() {
		if userEdit.Text() == "" {
			walk.MsgBox(owner, "警告", "请输入用户名！", walk.MsgBoxIconWarning)
			return
		}
		if pwdEdit.Text() == "" {
			walk.MsgBox(owner, "警告", "请输入密码！", walk.MsgBoxIconWarning)
			return
		}
		result.Account = config.Account{
			Username:     userEdit.Text(),
			Password:     pwdEdit.Text(),
			SavePassword: true,
			AutoLogin:    true,
			LastCode:     acc.LastCode,
			LastConpwd:   acc.LastConpwd,
		}
		result.Accepted = true
		dlg.Close(walk.DlgCmdOK)
	})

	cancelBtn, _ := walk.NewPushButton(btnComposite)
	cancelBtn.SetText("取消")
	cancelBtn.Clicked().Attach(func() {
		dlg.Close(walk.DlgCmdCancel)
	})

	dlg.Run()
	return result
}
