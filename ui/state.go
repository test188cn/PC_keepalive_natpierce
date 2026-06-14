package ui

// ConnMode represents the connection type.
type ConnMode int

const (
	ConnModeHost ConnMode = iota
	ConnModeCode
)

// HostInfo stores information about a remote host.
type HostInfo struct {
	ID       string
	Name     string
	IP       string
	Mappings string
	Mode     string
}

// AppState holds all application state flags.
type AppState struct {
	Connected      bool
	LoggedIn       bool
	LoggedInUser   string // username of the logged-in account
	PendingLoginUser string // username sent in login request, used for login success
	CurrentPCID    string
	ConnectionType ConnMode
	Reconnecting   bool

	KeepaliveRunning  bool
	KeepaliveConnMode int // connection mode at keepalive start: 0=host, 1=code
	LastSelectedPCID  string
	LastUsedCode      string

	Hosts       []HostInfo
	PCInfoCount int
}

// NewState creates a new AppState with default values.
func NewState() *AppState {
	return &AppState{
		ConnectionType: ConnModeHost,
	}
}

// IsCodeConnected returns true if connected via identification code.
func (s *AppState) IsCodeConnected() bool {
	return s.CurrentPCID == "code_connected" || s.ConnectionType == ConnModeCode
}
