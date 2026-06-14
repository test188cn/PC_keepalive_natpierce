package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"GoWork/config"
	"GoWork/protocol"
	"GoWork/sign"
	"GoWork/wsclient"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// App holds the entire application state and widgets.
type App struct {
	cfg   *config.Config
	state *AppState
	ws    *wsclient.Client
	ka    *Keepalive

	mainWindow      *walk.MainWindow
	startBtn        *walk.PushButton
	statusLbl       *walk.Label
	accCombo        *walk.ComboBox
	currentAccLabel *walk.Label
	hotSwitchBtn    *walk.PushButton
	connectionMode  *walk.ComboBox
	hostCombo       *walk.ComboBox
	refreshBtn      *walk.PushButton
	codeEdit        *walk.LineEdit
	conpwdEdit      *walk.LineEdit
	hostPanel       *walk.GroupBox
	codePanel       *walk.GroupBox
	kaSpin          *walk.NumberEdit
	keepaliveBtn    *walk.PushButton
	disconnectBtn   *walk.PushButton
	hostInfoLabel   *walk.Label
	logTextEdit     *walk.TextEdit
	logToggleBtn    *walk.PushButton
	logCollapsed    bool
	svcProcess      *exec.Cmd
	initialized     bool // true after first account load
	notifyIcon      *walk.NotifyIcon
}

// NewApp creates a new application instance.
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:   cfg,
		state: NewState(),
		ka:    NewKeepalive(),
	}
}

// postAsync posts a function to be executed on the main thread without blocking the caller.
func (a *App) postAsync(fn func()) {
	go func() {
		a.mainWindow.Synchronize(fn)
	}()
}

// Run creates the main window and starts the application.
func (a *App) Run() int {
	err := MainWindow{
		AssignTo: &a.mainWindow,
		Title:    "皎月连守护 v1.1",
		MinSize:  Size{Width: 580, Height: 420},
		Size:     Size{Width: 600, Height: 440},
		Layout:   VBox{Margins: Margins{Left: 8, Top: 6, Right: 8, Bottom: 6}, Spacing: 4},
		Children: []Widget{
			// Toolbar
			Composite{
				Layout: HBox{Spacing: 6},
				Children: []Widget{
					PushButton{
						Text:      "⚙️ 设置",
						MinSize:   Size{Width: 60, Height: 26},
						OnClicked: a.openSettings,
					},
					PushButton{
						AssignTo:  &a.startBtn,
						Text:      "🚀 启动服务",
						MinSize:   Size{Width: 80, Height: 26},
						OnClicked: a.startService,
					},
					PushButton{
						Text:      "📝 每日签到",
						MinSize:   Size{Width: 60, Height: 26},
						OnClicked: a.batchSignIn,
					},
					HSpacer{},
					Label{
						AssignTo:  &a.statusLbl,
						Text:      "● 未启动",
						TextColor: walk.RGB(128, 128, 128),
					},
				},
			},

			// Account Management
			GroupBox{
				Title:  "账户管理",
				Layout: VBox{Spacing: 4},
				Children: []Widget{
					// Row 1: account selector + add/edit/delete
					Composite{
						Layout: HBox{Spacing: 4},
						Children: []Widget{
							Label{Text: "账户:"},
							ComboBox{
								AssignTo:              &a.accCombo,
								Model:                 []string{"-- 手动输入 --"},
								CurrentIndex:          0,
								OnCurrentIndexChanged: a.onAccountSwitched,
							},
							PushButton{
								Text:      "➕ 添加",
								MinSize:   Size{Width: 50, Height: 24},
								OnClicked: a.addAccount,
							},
							PushButton{
								Text:      "✏️ 编辑",
								MinSize:   Size{Width: 50, Height: 24},
								OnClicked: a.editAccount,
							},
							PushButton{
								Text:      "🗑️ 删除",
								MinSize:   Size{Width: 50, Height: 24},
								OnClicked: a.deleteAccount,
							},
						},
					},
					// Row 2: current account display + switch button
					Composite{
						Layout: HBox{Spacing: 4},
						Children: []Widget{
							Label{Text: "当前登录账号:"},
							Label{
								AssignTo: &a.currentAccLabel,
								Text:     "-- 未登录 --",
							},
							HSpacer{},
							PushButton{
								AssignTo:  &a.hotSwitchBtn,
								Text:      "账户切换",
								MinSize:   Size{Width: 60, Height: 24},
								Enabled:   false,
								OnClicked: a.hotSwitch,
							},
						},
					},
				},
			},

			// Connection Mode
			ComboBox{
				AssignTo: &a.connectionMode,
				Model:    []string{"🌐 主机列表连接", "🔑 识别码连接"},
				OnCurrentIndexChanged: func() {
					idx := a.connectionMode.CurrentIndex()
					a.hostPanel.SetVisible(idx == 0)
					a.codePanel.SetVisible(idx == 1)
				},
			},

			// Host Panel
			GroupBox{
				AssignTo: &a.hostPanel,
				Title:    "主机列表",
				Layout:   HBox{Spacing: 5},
				Children: []Widget{
					ComboBox{
						AssignTo: &a.hostCombo,
						Model:    []string{"-- 选择主机 --"},
					},
					PushButton{
						AssignTo:  &a.refreshBtn,
						Text:      "🔄 刷新",
						MinSize:   Size{Width: 50, Height: 24},
						Enabled:   false,
						OnClicked: a.refreshHosts,
					},
				},
			},

			// Code Panel
			GroupBox{
				AssignTo: &a.codePanel,
				Title:    "识别码连接",
				Layout:   VBox{},
				Visible:  false,
				Children: []Widget{
					Composite{
						Layout: HBox{Spacing: 10},
						Children: []Widget{
							Label{
								Text:    "识别码:",
								MinSize: Size{Width: 50, Height: 24},
							},
							LineEdit{
								AssignTo:  &a.codeEdit,
								CueBanner: "输入识别码",
								MaxLength: 8,
								MinSize:   Size{Width: 120, Height: 24},
							},
							Label{
								Text:    "密码:",
								MinSize: Size{Width: 40, Height: 24},
							},
							LineEdit{
								AssignTo:     &a.conpwdEdit,
								CueBanner:    "输入连接密码",
								PasswordMode: true,
								MinSize:      Size{Width: 160, Height: 24},
							},
						},
					},
				},
			},

			// Keepalive Controls
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "保持(秒):"},
					NumberEdit{
						AssignTo:           &a.kaSpin,
						MinValue:           0,
						MaxValue:           86400,
						Value:              float64(a.cfg.KeepAlive),
						Decimals:           0,
						SpinButtonsVisible: true,
					},
					PushButton{
						AssignTo:  &a.keepaliveBtn,
						Text:      "🔁 保活",
						MinSize:   Size{Width: 60, Height: 26},
						Enabled:   false,
						OnClicked: a.toggleKeepalive,
					},
					PushButton{
						AssignTo:  &a.disconnectBtn,
						Text:      "❌ 断开",
						MinSize:   Size{Width: 60, Height: 26},
						Enabled:   false,
						OnClicked: a.disconnect,
					},
				},
			},

			// Host Info
			Label{
				AssignTo: &a.hostInfoLabel,
				Text:     "",
				Visible:  false,
			},

			// Log Panel
			Composite{
				Layout: VBox{Margins: Margins{Left: 3, Top: 3, Right: 3, Bottom: 3}, Spacing: 0},
				Children: []Widget{
					PushButton{
						AssignTo:  &a.logToggleBtn,
						Text:      "▶ 日志（点击展开）",
						MinSize:   Size{Height: 20},
						OnClicked: a.toggleLog,
					},
					TextEdit{
						AssignTo: &a.logTextEdit,
						ReadOnly: true,
						Visible:  false,
						MinSize:  Size{Height: 130},
						MaxSize:  Size{Height: 130},
					},
				},
			},
		},
	}.Create()

	if err != nil {
		os.WriteFile("crash.log", []byte(fmt.Sprintf("创建窗口失败: %v", err)), 0644)
		return 1
	}

	a.loadAccountsToUI()
	a.initialized = true

	// Enable vertical scrollbar on log TextEdit via Windows API
	win.SetWindowLong(a.logTextEdit.Handle(), win.GWL_STYLE,
		win.GetWindowLong(a.logTextEdit.Handle(), win.GWL_STYLE)|win.WS_VSCROLL)

	// Set window/taskbar icon using man.ico
	a.setupWindowIcon()

	// Setup system tray icon
	a.setupTrayIcon()

	DelayedFunc(1*time.Second, func() {
		a.postAsync(func() {
			a.autoLoginOnStartup()
		})
	})

	return a.mainWindow.Run()
}

// setupWindowIcon sets the window and taskbar icon using embedded man.ico.
func (a *App) setupWindowIcon() {
	tmpIcon := filepath.Join(os.TempDir(), "gowork_window.ico")
	if err := os.WriteFile(tmpIcon, windowIconData, 0644); err == nil {
		if hIcon, err := loadHIconFromFile(tmpIcon); err == nil {
			// Set both small (taskbar/title bar) and large (Alt+Tab) icon
			win.SendMessage(a.mainWindow.Handle(), win.WM_SETICON, 0, uintptr(hIcon)) // ICON_BIG
			win.SendMessage(a.mainWindow.Handle(), win.WM_SETICON, 1, uintptr(hIcon)) // ICON_SMALL
		}
		os.Remove(tmpIcon)
	}
}

// setupTrayIcon creates the system tray icon with context menu.
func (a *App) setupTrayIcon() {
	var err error
	a.notifyIcon, err = walk.NewNotifyIcon(a.mainWindow)
	if err != nil {
		a.appendLog(fmt.Sprintf("⚠️ 创建托盘图标失败: %v", err))
		return
	}

	a.notifyIcon.SetToolTip("皎月连守护 v1.1")

	// Write embedded tray icon to temp file, load with Windows API
	tmpIcon := filepath.Join(os.TempDir(), "gowork_tray.ico")
	if err := os.WriteFile(tmpIcon, trayIconData, 0644); err == nil {
		if hIcon, err := loadHIconFromFile(tmpIcon); err == nil {
			// Use Walk's UID (0) and CbSize format to match Walk's NotifyIcon
			var nid win.NOTIFYICONDATA
			nid.CbSize = uint32(unsafe.Sizeof(nid) - unsafe.Sizeof(win.HICON(0)))
			nid.HWnd = a.mainWindow.Handle()
			nid.UFlags = 2 // NIF_ICON
			nid.HIcon = hIcon
			win.Shell_NotifyIcon(1, &nid) // NIM_MODIFY
			a.appendLog("✅ 托盘图标已设置")
		} else {
			a.appendLog(fmt.Sprintf("⚠️ 图标加载失败: %v", err))
		}
		os.Remove(tmpIcon)
	} else {
		a.appendLog(fmt.Sprintf("⚠️ 临时图标文件写入失败: %v", err))
	}

	// Context menu
	menu := a.notifyIcon.ContextMenu()

	showAction := walk.NewAction()
	showAction.SetText("显示主界面")
	showAction.Triggered().Attach(func() {
		a.mainWindow.Show()
		a.mainWindow.SetFocus()
	})
	menu.Actions().Add(showAction)

	menu.Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText("退出软件")
	exitAction.Triggered().Attach(func() {
		a.state.KeepaliveRunning = false
		a.Cleanup()
		walk.App().Exit(0)
	})
	menu.Actions().Add(exitAction)

	a.notifyIcon.SetVisible(true)

	// Close button → minimize to tray
	a.mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		a.mainWindow.Hide()
	})

	// Double-click tray icon → restore
	a.notifyIcon.MessageClicked().Attach(func() {
		a.mainWindow.Show()
		a.mainWindow.SetFocus()
	})
}

// autoLoginOnStartup loads the last used account and starts connection.
func (a *App) autoLoginOnStartup() {
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 {
		a.appendLog("⚠️ 未选择账户，跳过自动登录")
		return
	}
	if idx-1 >= len(a.cfg.Accounts) {
		a.appendLog("❌ 账户不存在或未添加")
		return
	}
	acc := a.cfg.Accounts[idx-1]
	if acc.Username == "" {
		a.appendLog("❌ 账户用户名为空")
		return
	}
	if acc.Password == "" {
		a.appendLog(fmt.Sprintf("❌ 账户 %s 密码为空，请编辑账户补充密码", acc.Username))
		return
	}
	a.appendLog(fmt.Sprintf("🚀 启动服务，账户: %s", acc.Username))
	a.startService()
}

func (a *App) startService() {
	exePath := a.cfg.ExePath
	if exePath != "" {
		a.appendLog(fmt.Sprintf("🚀 启动服务: %s", exePath))
		cmd := exec.Command(exePath)
		cmd.Dir = filepath.Dir(exePath)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err != nil {
			a.appendLog(fmt.Sprintf("❌ 启动服务失败: %v", err))
			return
		}
		a.svcProcess = cmd
		a.appendLog(fmt.Sprintf("✅ 服务进程已启动 (PID: %d)", cmd.Process.Pid))
		DelayedFunc(3*time.Second, func() {
			a.postAsync(func() {
				a.connectAfterStart()
			})
		})
	} else {
		a.appendLog("🔗 直接连接服务端...")
		a.connectAfterStart()
	}
}

func (a *App) connectAfterStart() {
	url := a.cfg.ServerURL
	if url == "" {
		a.appendLog("⚠️ 未配置服务地址，请先在设置中填写")
		return
	}

	a.appendLog("🔗 连接服务端...")

	if a.ws != nil {
		a.ws.Close()
	}

	a.ws = wsclient.New(a.onWSEvent)

	err := a.ws.Connect(url)
	if err != nil {
		a.appendLog(fmt.Sprintf("❌ 连接失败: %v", err))
		return
	}
	// Login will be triggered in EventConnected handler
}

func (a *App) sendLoginMsg(user, pwd string) {
	a.state.PendingLoginUser = user
	// Find account settings for save/auto
	save, auto := false, false
	for _, acc := range a.cfg.Accounts {
		if acc.Username == user {
			save = acc.SavePassword
			auto = acc.AutoLogin
			break
		}
	}
	msg := protocol.MsgLogin(user, pwd, save, auto)
	if a.ws.Send(msg) {
		a.appendLog(fmt.Sprintf("🔐 登录请求已发送 (%s)", user))
	}
}

// autoLogin sends login message using the selected account credentials.
func (a *App) autoLogin() {
	// First try the currently selected account from combo box
	idx := a.accCombo.CurrentIndex()
	if idx > 0 && idx-1 < len(a.cfg.Accounts) {
		acc := a.cfg.Accounts[idx-1]
		if acc.Username != "" && acc.Password != "" {
			a.appendLog(fmt.Sprintf("🔐 自动登录: %s", acc.Username))
			a.sendLoginMsg(acc.Username, acc.Password)
			return
		}
	}
	// Fallback: use first account with credentials
	for _, acc := range a.cfg.Accounts {
		if acc.Username != "" && acc.Password != "" {
			a.appendLog(fmt.Sprintf("🔐 自动登录: %s", acc.Username))
			a.sendLoginMsg(acc.Username, acc.Password)
			return
		}
	}
	a.appendLog("⚠️ 无已保存账户，请通过「添加」按钮添加账户后选择登录")
}

func (a *App) onWSEvent(evt wsclient.Event) {
	a.postAsync(func() {
		switch evt.Type {
		case wsclient.EventConnected:
			a.state.Connected = true
			a.appendLog("✅ WebSocket已连接")
			a.setStatus("● 已连接", "ok")
			a.setButtons(true)
			// Auto-login after successful connection (only if not already logged in)
			if !a.state.LoggedIn {
				a.autoLogin()
			}
		case wsclient.EventDisconnected:
			a.state.Connected = false
			a.state.LoggedIn = false
			a.state.LoggedInUser = ""
			a.state.CurrentPCID = ""
			a.state.ConnectionType = ConnModeHost
			a.setStatus("● 已断开", "off")
			a.disconnectBtn.SetEnabled(false)
			a.setHostInfo("")
			a.currentAccLabel.SetText("-- 未登录 --")
			a.appendLog(fmt.Sprintf("⚠️ WebSocket断开: %s", evt.Reason))
			// Always auto-reconnect WebSocket
			a.appendLog("🔁 3秒后自动重连服务端...")
			DelayedFunc(3*time.Second, func() {
				a.postAsync(func() {
					a.autoReconnectWS()
				})
			})
		case wsclient.EventMessage:
			a.onWSMessage(evt.Message)
		}
	})
}

func (a *App) onWSMessage(raw string) {
	resp := protocol.Parse(raw)
	switch resp.Type {
	case "start":
		version := resp.Field(1)
		if version == "" {
			version = "?"
		}
		a.appendLog(fmt.Sprintf("🖥️ 服务端: v%s", version))
	case "0":
		errMsg := resp.Field(5)
		if errMsg != "" {
			a.appendLog(fmt.Sprintf("❌ 登录失败: %s", errMsg))
		}
	case "1", "2":
		// Guard against duplicate login success messages
		if a.state.LoggedIn {
			a.appendLog("⚠️ 忽略重复的登录成功消息")
			return
		}
		a.state.LoggedIn = true
		a.state.LoggedInUser = a.state.PendingLoginUser
		if resp.Type == "1" {
			a.appendLog("✅ 登录成功 (组网服务未启动)")
		} else {
			a.appendLog("✅ 登录成功 (服务已启动)")
		}
		// Clear old host data before loading new account's hosts
		a.state.Hosts = nil
		a.state.PCInfoCount = 0
		a.hostCombo.SetModel([]string{"-- 选择主机 --"})
		a.setHostInfo("")
		a.activateAfterLogin()
		// Delay pclist request to avoid overwhelming the server
		DelayedFunc(500*time.Millisecond, func() {
			a.postAsync(func() {
				if a.ws != nil && a.state.Connected {
					a.ws.Send(protocol.MsgPclist())
				}
			})
		})
		if a.state.KeepaliveRunning && a.state.Reconnecting {
			DelayedFunc(2*time.Second, func() {
				a.postAsync(func() {
					a.reconnectHostAfterLogin()
				})
			})
		}
	case "pclist":
		a.handlePCList(resp)
	case "pcinfo":
		a.handlePCInfo(resp)
	case "conpc":
		a.handleConPC(resp)
	case "conerr":
		a.state.CurrentPCID = ""
		a.state.ConnectionType = ConnModeHost
		a.disconnectBtn.SetEnabled(false)
		errMsg := resp.Field(1)
		if errMsg == "" {
			errMsg = "未知"
		}
		a.appendLog(fmt.Sprintf("❌ 连接错误: %s", errMsg))
		if a.state.KeepaliveRunning {
			a.appendLog("⚠️ 连接错误，终止保活")
			a.stopKeepalive()
		}
	case "discon":
		// Always clear connection state on disconnect, even during reconnect
		a.state.CurrentPCID = ""
		a.state.ConnectionType = ConnModeHost
		if !a.state.Reconnecting {
			a.setStatus("● 已登录", "ok")
			a.disconnectBtn.SetEnabled(false)
			a.setHostInfo("")
			a.appendLog("🔌 主机连接已断开")
			if a.state.KeepaliveRunning && (a.state.LastSelectedPCID != "" || a.state.LastUsedCode != "") {
				a.appendLog("🔁 保活: 主机断开，自动重连...")
				DelayedFunc(1*time.Second, func() {
					a.postAsync(func() {
						a.reconnectHostAfterLogin()
					})
				})
			}
		}
	case "info":
		msg := resp.Field(1)
		if msg != "" {
			a.appendLog(fmt.Sprintf("ℹ️ %s", msg))
		}
	case "tips":
		msg := resp.Field(1)
		if msg != "" {
			a.appendLog(fmt.Sprintf("💡 %s", msg))
		}
	case "log":
		msg := resp.Field(4)
		if msg != "" {
			a.appendLog(fmt.Sprintf("📝 %s", msg))
		}
	case "vipend":
		a.appendLog("⛔ 服务已到期，请官网签到或续费后继续使用")
		// Auto-trigger batch sign-in
		a.appendLog("📝 检测到服务到期，自动触发批量签到...")
		DelayedFunc(2*time.Second, func() {
			a.postAsync(func() {
				a.batchSignIn()
			})
		})
	}
}

func (a *App) handlePCList(resp protocol.Response) {
	var hosts []HostInfo
	i := 1
	for i+3 < resp.FieldCount()-4 {
		hosts = append(hosts, HostInfo{
			ID: resp.Field(i), Name: resp.Field(i + 1),
			IP: resp.Field(i + 2), Mappings: resp.Field(i + 3), Mode: "⏳ 获取中...",
		})
		i += 4
	}
	a.state.Hosts = hosts
	a.state.PCInfoCount = 0
	a.hostCombo.SetModel(formatHostList(hosts))
	if a.state.Connected {
		for idx := range hosts {
			pcid := a.state.Hosts[idx].ID
			DelayedFunc(time.Duration(idx*100)*time.Millisecond, func() {
				a.ws.Send(protocol.MsgPCInfo(pcid))
			})
		}
	}
	a.appendLog(fmt.Sprintf("📋 在线主机: %d 台", len(hosts)))
}

func (a *App) handlePCInfo(resp protocol.Response) {
	if resp.FieldCount() < 5 {
		return
	}
	a.state.PCInfoCount++
	pcid, ip, pcname, ms := resp.Field(1), resp.Field(2), resp.Field(3), resp.Field(4)
	mode := detectMode(ms)
	// Only show host info when keepalive is running
	if a.state.KeepaliveRunning {
		a.setHostInfo(fmt.Sprintf("主机: %s | IP: %s | 模式: %s", pcname, ip, mode))
	}
	for i, h := range a.state.Hosts {
		if h.ID == pcid {
			a.state.Hosts[i].Mode = mode
			a.state.Hosts[i].IP = ip
			break
		}
	}
	a.updateComboDisplay(pcid, pcname, ip, mode)
	a.appendLog(fmt.Sprintf("✅ [%d/%d] %s: %s (%s)", a.state.PCInfoCount, len(a.state.Hosts), pcname, mode, ip))
}

func (a *App) handleConPC(resp protocol.Response) {
	ok := resp.Field(1) == "yes"
	pid, ip := resp.Field(2), resp.Field(3)
	if ok {
		isCodeConn := a.state.LastUsedCode != "" && a.connectionMode.CurrentIndex() == 1
		if isCodeConn {
			a.state.CurrentPCID = "code_connected"
			a.state.ConnectionType = ConnModeCode
		} else {
			a.state.CurrentPCID = pid
			a.state.ConnectionType = ConnModeHost
		}
		a.state.Reconnecting = false
		a.setStatus("● 已连接", "run")
		a.disconnectBtn.SetEnabled(true)

		// Single host lookup
		hostName := pid
		hostMode := ""
		for _, h := range a.state.Hosts {
			if h.ID == pid {
				hostName = h.Name
				hostMode = h.Mode
				break
			}
		}

		// Build host info display (only when keepalive is running)
		if a.state.KeepaliveRunning {
			if isCodeConn {
				a.setHostInfo(fmt.Sprintf("识别码: %s | IP: 127.0.0.1", a.state.LastUsedCode))
			} else if strings.Contains(hostMode, "组网") {
				displayIP := ip
				if displayIP == "" {
					displayIP = "127.0.0.1"
				}
				a.setHostInfo(fmt.Sprintf("主机: %s | IP: %s | 模式: 🌐 组网", hostName, displayIP))
			} else {
				a.setHostInfo(fmt.Sprintf("主机: %s | IP: 127.0.0.1 | 模式: 🔀 映射", hostName))
			}
		}

		// Log connection success
		if isCodeConn {
			a.appendLog(fmt.Sprintf("🎉 连接成功！(识别码: %s)", a.state.LastUsedCode))
		} else {
			a.appendLog(fmt.Sprintf("🎉 连接成功！(主机: %s)", hostName))
		}
		if a.state.KeepaliveRunning {
			interval := time.Duration(max(5, int(a.kaSpin.Value()))) * time.Second
			a.ka.Restart(interval, a.keepaliveCheck)
		}
	} else {
		a.appendLog(fmt.Sprintf("❌ 连接失败: %s", pid))
		a.state.Reconnecting = false
		a.state.CurrentPCID = ""
		a.state.ConnectionType = ConnModeHost
		if a.state.KeepaliveRunning {
			a.appendLog("⚠️ 连接失败，终止保活")
			a.stopKeepalive()
		}
	}
}

func (a *App) activateAfterLogin() {
	a.setButtons(true)
	a.setStatus("● 已登录", "ok")
	a.connectionMode.SetCurrentIndex(0)
	// Show logged-in account name
	a.currentAccLabel.SetText(a.state.LoggedInUser)
	a.currentAccLabel.RequestLayout()
	// Load code/conpwd from the logged-in account
	for _, acc := range a.cfg.Accounts {
		if acc.Username == a.state.LoggedInUser {
			a.codeEdit.SetText(acc.LastCode)
			a.conpwdEdit.SetText(acc.LastConpwd)
			return
		}
	}
}

func (a *App) setButtons(loggedIn bool) {
	a.refreshBtn.SetEnabled(loggedIn)
	a.keepaliveBtn.SetEnabled(loggedIn)
	a.hotSwitchBtn.SetEnabled(loggedIn)
}

func (a *App) setHostInfo(text string) {
	if text == "" {
		a.hostInfoLabel.SetVisible(false)
	} else {
		a.hostInfoLabel.SetText(text)
		a.hostInfoLabel.SetVisible(true)
		a.hostInfoLabel.RequestLayout()
	}
}

// showCurrentHostInfo displays host info based on current connection state.
func (a *App) showCurrentHostInfo() {
	if a.state.IsCodeConnected() {
		a.setHostInfo(fmt.Sprintf("识别码: %s | IP: 127.0.0.1", a.state.LastUsedCode))
		return
	}
	if a.state.CurrentPCID == "" {
		return
	}
	for _, h := range a.state.Hosts {
		if h.ID == a.state.CurrentPCID {
			if strings.Contains(h.Mode, "组网") {
				displayIP := h.IP
				if displayIP == "" {
					displayIP = "127.0.0.1"
				}
				a.setHostInfo(fmt.Sprintf("主机: %s | IP: %s | 模式: 🌐 组网", h.Name, displayIP))
			} else {
				a.setHostInfo(fmt.Sprintf("主机: %s | IP: 127.0.0.1 | 模式: 🔀 映射", h.Name))
			}
			return
		}
	}
	// Fallback: host not found in list, show raw pcid
	a.setHostInfo(fmt.Sprintf("主机: %s | 连接中", a.state.CurrentPCID))
}

// batchSignIn signs in all accounts from config.
func (a *App) batchSignIn() {
	a.appendLog("📝 ===== 开始批量签到 =====")

	if len(a.cfg.Accounts) == 0 {
		a.appendLog("📝 ⚠️ 没有保存的账户，无法签到")
		return
	}

	a.appendLog(fmt.Sprintf("📝 共 %d 个账户待签到", len(a.cfg.Accounts)))

	// Run sign-in in goroutine to avoid blocking UI
	go func() {
		// Collect all log messages first, then output in order
		var allLogs []string
		for _, acc := range a.cfg.Accounts {
			logs := sign.SignInAccount(acc.Username, acc.Password)
			allLogs = append(allLogs, logs...)
		}
		allLogs = append(allLogs, "📝 ===== 批量签到完成 =====")

		// Post all logs in order
		a.postAsync(func() {
			for _, msg := range allLogs {
				a.appendLog(msg)
			}
		})
	}()
}

func (a *App) setStatus(text, style string) {
	a.statusLbl.SetText(text)
	colors := map[string][3]uint8{
		"ok":   {0, 128, 0},
		"err":  {255, 0, 0},
		"run":  {126, 4, 179},
		"wait": {255, 165, 0},
		"off":  {128, 128, 128},
	}
	if c, ok := colors[style]; ok {
		a.statusLbl.SetTextColor(walk.RGB(c[0], c[1], c[2]))
	}
	a.statusLbl.RequestLayout()
}

func (a *App) updateComboDisplay(pcid, pcname, ip, mode string) {
	items := a.hostCombo.Model().([]string)
	for i, h := range a.state.Hosts {
		if h.ID == pcid && (i+1) < len(items) {
			if ip != "" {
				items[i+1] = fmt.Sprintf("%s %s (%s)", pcname, mode, ip)
			} else {
				items[i+1] = fmt.Sprintf("%s %s", pcname, mode)
			}
			a.hostCombo.SetModel(items)
			return
		}
	}
}

func (a *App) openSettings() {
	result := ShowSettingsDialog(a.mainWindow, a.cfg.ExePath, a.cfg.ServerURL)
	if result.Accepted {
		a.cfg.ExePath = result.ExePath
		a.cfg.ServerURL = result.ServerURL
		a.cfg.Save()
		a.appendLog("⚙️ 设置已保存")
	}
}

func (a *App) refreshHosts() {
	if a.ws.Send(protocol.MsgPclist()) {
		a.appendLog("🔄 刷新主机列表...")
	}
}

func (a *App) disconnect() {
	if a.ka.IsRunning() {
		a.stopKeepalive()
		a.appendLog("⏹️ 已停止保活")
		return
	}
	if a.state.CurrentPCID != "" {
		a.ws.Send(protocol.MsgStopCon())
		a.state.CurrentPCID = ""
		a.state.ConnectionType = ConnModeHost
		a.appendLog("🔌 已断开连接")
		a.setStatus("● 已登录", "ok")
		a.disconnectBtn.SetEnabled(false)
		a.setHostInfo("")
	} else {
		a.appendLog("⚠️ 当前未连接主机或识别码")
	}
}

func (a *App) toggleKeepalive() {
	if a.ka.IsRunning() {
		a.stopKeepalive()
	} else {
		a.startKeepalive()
	}
}

func (a *App) startKeepalive() {
	if a.ka.IsRunning() {
		a.appendLog("⚠️ 保活已在运行中")
		return
	}
	if !a.state.Connected {
		a.appendLog("🔗 保活：连接服务端...")
		a.connectAfterStart()
		DelayedFunc(5*time.Second, func() {
			a.postAsync(func() {
				a.doKeepaliveConnect()
			})
		})
		return
	}
	a.doKeepaliveConnect()
}

func (a *App) doKeepaliveConnect() {
	if !a.state.Connected || !a.state.LoggedIn {
		a.appendLog("⚠️ 未连接到服务端")
		return
	}
	connMode := a.connectionMode.CurrentIndex()
	if connMode == 0 {
		// Try combo selection first, then fall back to LastSelectedPCID
		idx := a.hostCombo.CurrentIndex()
		var pcid string
		if idx > 0 && idx <= len(a.state.Hosts) {
			pcid = a.state.Hosts[idx-1].ID
		} else if a.state.LastSelectedPCID != "" {
			// Find pcid in hosts list
			for _, h := range a.state.Hosts {
				if h.ID == a.state.LastSelectedPCID {
					pcid = h.ID
					break
				}
			}
			if pcid == "" && len(a.state.Hosts) > 0 {
				pcid = a.state.Hosts[0].ID
			}
		} else if len(a.state.Hosts) > 0 {
			pcid = a.state.Hosts[0].ID
		}
		if pcid == "" {
			walk.MsgBox(a.mainWindow, "提示", "请先选择主机！", walk.MsgBoxIconWarning)
			return
		}
		a.state.LastSelectedPCID = pcid
		a.ws.Send(protocol.MsgConPC(pcid))
		a.appendLog(fmt.Sprintf("🔁 保活: 连接主机 (pcid=%s)", pcid))
	} else {
		code := a.codeEdit.Text()
		pwd := a.conpwdEdit.Text()
		if code == "" || len(code) != 8 {
			walk.MsgBox(a.mainWindow, "提示", "请输入8位识别码！", walk.MsgBoxIconWarning)
			return
		}
		a.state.LastUsedCode = code
		a.ws.Send(protocol.MsgConCode(code, pwd))
		a.appendLog(fmt.Sprintf("🔁 保活: 通过识别码连接 (%s)", code))
		a.saveCurrentCodeToAccount()
	}
	a.keepaliveBtn.SetText("⏹️ 停止保活")
	a.state.KeepaliveRunning = true
	a.state.Reconnecting = true
	a.state.KeepaliveConnMode = connMode // record mode at keepalive start
	interval := time.Duration(max(5, int(a.kaSpin.Value()))) * time.Second
	a.appendLog(fmt.Sprintf("⏱️ 保活模式: 每 %d 秒检查一次", int(interval.Seconds())))
	a.ka.Start(interval, a.keepaliveCheck)
}

func (a *App) keepaliveCheck() {
	a.postAsync(func() {
		if !a.state.KeepaliveRunning || a.state.Reconnecting {
			return
		}
		if a.state.CurrentPCID == "" && a.state.ConnectionType == ConnModeHost {
			a.appendLog("⏳ 等待连接成功...")
			return
		}
		currentOK := false
		connMode := a.state.KeepaliveConnMode // use saved mode, not live dropdown
		if connMode == 0 {
			currentOK = a.state.CurrentPCID == a.state.LastSelectedPCID
		} else {
			currentOK = a.state.IsCodeConnected() && a.state.LastUsedCode != ""
		}
		if !currentOK {
			a.appendLog("⚠️ 连接配置已改变，重新连接...")
			a.reconnectHostAfterLogin()
			return
		}
		// Show host info on first successful keepalive check
		if a.hostInfoLabel.Visible() == false {
			a.showCurrentHostInfo()
		}
		a.appendLog("✅ 保活检查: 正常连接中")
	})
}

// autoReconnectWS reconnects the WebSocket and re-logs in.
// If keepalive is running, it also reconnects to the host/code after login.
func (a *App) autoReconnectWS() {
	if a.state.Connected {
		return // already connected, no need
	}

	a.appendLog("🔄 重新连接服务端...")
	if a.ws != nil {
		a.ws.Close()
	}

	url := a.cfg.ServerURL

	// Get credentials from current account
	var user, pwd string
	idx := a.accCombo.CurrentIndex()
	if idx > 0 && idx-1 < len(a.cfg.Accounts) {
		acc := a.cfg.Accounts[idx-1]
		user = acc.Username
		pwd = acc.Password
	}

	if url == "" || user == "" || pwd == "" {
		a.appendLog("⚠️ 服务地址/用户名/密码为空，无法重连")
		return
	}

	a.ws = wsclient.New(a.onWSEvent)
	err := a.ws.Connect(url)
	if err != nil {
		a.appendLog(fmt.Sprintf("❌ 重连失败: %v", err))
		return
	}

	DelayedFunc(1*time.Second, func() {
		a.postAsync(func() {
			a.sendLoginMsg(user, pwd)
		})
	})

	// After login, reconnect host/code if keepalive is running
	if a.state.KeepaliveRunning {
		DelayedFunc(3*time.Second, func() {
			a.postAsync(func() {
				a.reconnectHostAfterLogin()
			})
		})
	}
}

func (a *App) reconnectHostAfterLogin() {
	if !a.state.KeepaliveRunning {
		return
	}
	if !a.state.Connected {
		a.appendLog("⚠️ WebSocket已断开，需要重新连接服务端...")
		a.autoReconnectWS()
		return
	}
	a.state.Reconnecting = true
	// Clear connection state immediately so keepidle check doesn't misfire
	a.state.CurrentPCID = ""
	a.state.ConnectionType = ConnModeHost
	a.disconnectBtn.SetEnabled(false)
	a.setHostInfo("")
	if a.state.LastSelectedPCID != "" || a.state.LastUsedCode != "" {
		a.ws.Send(protocol.MsgStopCon())
		a.appendLog("🔌 先断开旧连接")
	}
	DelayedFunc(1*time.Second, func() {
		a.postAsync(func() {
			a.doReconnect()
		})
	})
}

func (a *App) doReconnect() {
	if !a.state.KeepaliveRunning {
		return
	}
	connMode := a.connectionMode.CurrentIndex()
	if connMode == 0 && a.state.LastSelectedPCID != "" {
		a.ws.Send(protocol.MsgConPC(a.state.LastSelectedPCID))
		a.appendLog(fmt.Sprintf("🔗 重新连接主机 (pcid=%s)", a.state.LastSelectedPCID))
	} else if connMode == 1 && a.state.LastUsedCode != "" {
		pwd := a.conpwdEdit.Text()
		a.ws.Send(protocol.MsgConCode(a.state.LastUsedCode, pwd))
		a.appendLog("🔑 重新通过识别码连接")
		a.saveCurrentCodeToAccount()
	}
	DelayedFunc(10*time.Second, func() {
		a.postAsync(func() {
			if a.state.Reconnecting && a.state.KeepaliveRunning {
				a.state.Reconnecting = false
				interval := time.Duration(max(5, int(a.kaSpin.Value()))) * time.Second
				a.ka.Restart(interval, a.keepaliveCheck)
			}
		})
	})
}

func (a *App) stopKeepalive() {
	a.state.KeepaliveRunning = false
	a.ka.Stop()
	a.keepaliveBtn.SetText("🔁 保活")
	a.appendLog("⏹️ 保活已停止")
	if a.state.CurrentPCID != "" {
		a.ws.Send(protocol.MsgStopCon())
		a.state.CurrentPCID = ""
		a.state.ConnectionType = ConnModeHost
		a.appendLog("🔌 已断开连接")
	}
	a.setStatus("● 已登录", "ok")
	a.disconnectBtn.SetEnabled(false)
	a.setHostInfo("")
}

func (a *App) hotSwitch() {
	// Get credentials from selected account
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 || idx-1 >= len(a.cfg.Accounts) {
		walk.MsgBox(a.mainWindow, "提示", "请先选择账户！", walk.MsgBoxIconWarning)
		return
	}
	acc := a.cfg.Accounts[idx-1]
	user, pwd := acc.Username, acc.Password
	if user == "" || pwd == "" {
		walk.MsgBox(a.mainWindow, "提示", "账户用户名或密码为空！", walk.MsgBoxIconWarning)
		return
	}
	if !a.state.Connected {
		walk.MsgBox(a.mainWindow, "提示", "未连接服务器！", walk.MsgBoxIconWarning)
		return
	}
	if a.state.LoggedIn {
		result := walk.MsgBox(a.mainWindow, "热切换", fmt.Sprintf("当前已登录\n将切换到: %s\n确定继续？", user), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
		if result != walk.DlgCmdYes {
			return
		}
		a.appendLog(fmt.Sprintf("🔄 热切换 → %s...", user))
		a.setStatus("● 切换中...", "wait")
		a.disconnectBtn.SetEnabled(false)
		a.keepaliveBtn.SetEnabled(false)
		// Stop keepalive to prevent it from interfering with account switch
		if a.state.KeepaliveRunning {
			a.state.KeepaliveRunning = false
			a.ka.Stop()
			a.appendLog("⏹️ 保活已停止")
		}
		if a.state.CurrentPCID != "" {
			a.ws.Send(protocol.MsgStopCon())
			a.appendLog("🔌 已发送断开主机命令")
		}
		a.ws.Send(protocol.MsgExit())
		a.appendLog("📤 已发送退出登录命令")
		// Reset login state so new login success is not treated as duplicate
		a.state.LoggedIn = false
		a.state.LoggedInUser = ""
		a.currentAccLabel.SetText("-- 未登录 --")
		// Clear old host list
		a.state.Hosts = nil
		a.hostCombo.SetModel([]string{"-- 选择主机 --"})
		a.setHostInfo("")
		DelayedFunc(1500*time.Millisecond, func() {
			a.postAsync(func() {
				a.state.LoggedIn = false // Clear again before new login
				a.sendLoginMsg(user, pwd)
			})
		})
	} else {
		result := walk.MsgBox(a.mainWindow, "热切换", fmt.Sprintf("切换到账户: %s\n确定？", user), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
		if result != walk.DlgCmdYes {
			return
		}
		a.appendLog(fmt.Sprintf("🔄 热切换 → %s...", user))
		a.sendLoginMsg(user, pwd)
	}
	a.cfg.LastAccountIdx = a.accCombo.CurrentIndex()
	a.cfg.Save()
}

func (a *App) loadAccountsToUI() {
	items := []string{"-- 选择登录账户 --"}
	for _, acc := range a.cfg.Accounts {
		items = append(items, acc.Username)
	}
	a.accCombo.SetModel(items)
	idx := a.cfg.LastAccountIdx
	if idx >= 0 && idx < len(items) {
		a.accCombo.SetCurrentIndex(idx)
	}
	a.onAccountSwitched()
}

func (a *App) onAccountSwitched() {
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 || idx-1 >= len(a.cfg.Accounts) {
		a.codeEdit.SetText("")
		a.conpwdEdit.SetText("")
		return
	}
	acc := a.cfg.Accounts[idx-1]
	a.codeEdit.SetText(acc.LastCode)
	a.conpwdEdit.SetText(acc.LastConpwd)

	// If WebSocket is connected but not logged in, trigger login with selected account
	if a.initialized && a.state.Connected {
		if a.state.LoggedIn {
			// Already logged in - don't auto-login, user should use "账户切换" button
			return
		}
		if acc.Username != "" && acc.Password != "" {
			a.appendLog(fmt.Sprintf("🔐 选择账户: %s，正在登录...", acc.Username))
			a.sendLoginMsg(acc.Username, acc.Password)
		}
	}
}

// saveCurrentCodeToAccount saves the current code/conpwd to the current account in config.
func (a *App) saveCurrentCodeToAccount() {
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 || idx-1 >= len(a.cfg.Accounts) {
		return
	}
	a.cfg.Accounts[idx-1].LastCode = a.codeEdit.Text()
	a.cfg.Accounts[idx-1].LastConpwd = a.conpwdEdit.Text()
	a.cfg.Save()
}

func (a *App) addAccount() {
	result := ShowAccountDialog(a.mainWindow, false, config.Account{})
	if !result.Accepted {
		return
	}
	var accounts []config.Account
	for _, acc := range a.cfg.Accounts {
		if acc.Username != result.Account.Username {
			accounts = append(accounts, acc)
		}
	}
	accounts = append(accounts, result.Account)
	a.cfg.Accounts = accounts
	a.cfg.Save()
	a.loadAccountsToUI()
}

func (a *App) editAccount() {
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 || idx-1 >= len(a.cfg.Accounts) {
		walk.MsgBox(a.mainWindow, "提示", "请先选择账户", walk.MsgBoxIconInformation)
		return
	}
	acc := a.cfg.Accounts[idx-1]
	result := ShowAccountDialog(a.mainWindow, true, acc)
	if !result.Accepted {
		return
	}
	var accounts []config.Account
	for _, ac := range a.cfg.Accounts {
		if ac.Username != acc.Username {
			accounts = append(accounts, ac)
		}
	}
	accounts = append(accounts, result.Account)
	a.cfg.Accounts = accounts
	a.cfg.Save()
	a.loadAccountsToUI()
}

func (a *App) deleteAccount() {
	idx := a.accCombo.CurrentIndex()
	if idx <= 0 || idx-1 >= len(a.cfg.Accounts) {
		walk.MsgBox(a.mainWindow, "提示", "请先选择账户", walk.MsgBoxIconInformation)
		return
	}
	acc := a.cfg.Accounts[idx-1]
	result := walk.MsgBox(a.mainWindow, "确认", fmt.Sprintf("删除账户 %s？", acc.Username), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
	if result != walk.DlgCmdYes {
		return
	}
	var accounts []config.Account
	for _, ac := range a.cfg.Accounts {
		if ac.Username != acc.Username {
			accounts = append(accounts, ac)
		}
	}
	a.cfg.Accounts = accounts
	a.cfg.Save()
	a.loadAccountsToUI()
}

func (a *App) toggleLog() {
	a.logCollapsed = !a.logCollapsed
	if a.logCollapsed {
		a.logTextEdit.SetVisible(true)
		a.logToggleBtn.SetText("▼ 日志（点击折叠）")
	} else {
		a.logTextEdit.SetVisible(false)
		a.logToggleBtn.SetText("▶ 日志（点击展开）")
	}
}

func (a *App) appendLog(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s\r\n", ts, msg)
	text := a.logTextEdit.Text() + line

	// Keep only the last 50 lines
	lines := strings.Split(text, "\r\n")
	if len(lines) > 50 {
		lines = lines[len(lines)-50:]
	}
	a.logTextEdit.SetText(strings.Join(lines, "\r\n"))

	// Scroll to bottom
	a.logTextEdit.SendMessage(win.EM_LINESCROLL, 0, 99999)
}

func (a *App) Cleanup() {
	a.state.KeepaliveRunning = false
	if a.ka != nil {
		a.ka.Stop()
	}

	// Dispose tray icon
	if a.notifyIcon != nil {
		a.notifyIcon.SetVisible(false)
		a.notifyIcon.Dispose()
	}

	// Kill the service process started by this app
	if a.svcProcess != nil && a.svcProcess.Process != nil {
		a.appendLog(fmt.Sprintf("🛑 正在关闭服务进程 (PID: %d)...", a.svcProcess.Process.Pid))
		a.svcProcess.Process.Kill()
		a.svcProcess = nil
	} else if a.cfg.ExePath != "" {
		// Fallback: kill by EXE name if svcProcess reference was lost
		exeName := filepath.Base(a.cfg.ExePath)
		a.appendLog(fmt.Sprintf("🛑 正在关闭服务进程 (%s)...", exeName))
		killCmd := exec.Command("taskkill", "/F", "/IM", exeName)
		killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		killCmd.Run()
	}

	if a.ws != nil {
		a.ws.Close()
	}
	a.saveCurrentCodeToAccount()
	a.cfg.LastAccountIdx = a.accCombo.CurrentIndex()
	a.cfg.Save()
}

func formatHostList(hosts []HostInfo) []string {
	items := []string{"-- 选择主机 --"}
	for _, h := range hosts {
		if h.IP != "" {
			items = append(items, fmt.Sprintf("%s %s (%s)", h.Name, h.Mode, h.IP))
		} else {
			items = append(items, fmt.Sprintf("%s %s", h.Name, h.Mode))
		}
	}
	return items
}

func detectMode(ms string) string {
	if ms == "" {
		return "📭 无映射"
	}
	m := strings.Replace(ms, ";", "|", -1)
	parts := strings.Split(m, "|")
	if len(parts) >= 5 {
		if parts[3] == "all" && parts[2] == "all" {
			return "🌐 组网"
		} else if parts[3] != "" {
			return "🔀 映射"
		}
	}
	return "📭 无映射"
}
