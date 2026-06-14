package protocol

import "strings"

const SEP = "<$!$>"

// MsgLogin builds a login message.
func MsgLogin(user, pwd string, save, auto bool) string {
	return "login" + SEP + user + SEP + pwd + SEP + boolToInt(save) + SEP + boolToInt(auto)
}

// MsgPclist requests the host list.
func MsgPclist() string {
	return "pclist"
}

// MsgPCInfo requests info for a specific host.
func MsgPCInfo(pcid string) string {
	return "pcinfo" + SEP + pcid
}

// MsgConPC connects to a host by pcid.
func MsgConPC(pcid string) string {
	return "conpc" + SEP + pcid
}

// MsgConCode connects via identification code.
func MsgConCode(code, pwd string) string {
	return "concode" + SEP + code + SEP + pwd
}

// MsgStopCon stops the current connection.
func MsgStopCon() string {
	return "stopcon" + SEP
}

// MsgExit sends exit/logout command (for hot-switch).
func MsgExit() string {
	return "exit" + SEP
}

// Response represents a parsed server response.
type Response struct {
	Type   string
	Fields []string
}

// Parse splits a raw message by SEP and returns a Response.
func Parse(raw string) Response {
	fields := strings.Split(raw, SEP)
	t := ""
	if len(fields) > 0 {
		t = fields[0]
	}
	return Response{Type: t, Fields: fields}
}

// Field returns the field at index i, or empty string if out of bounds.
func (r Response) Field(i int) string {
	if i < 0 || i >= len(r.Fields) {
		return ""
	}
	return r.Fields[i]
}

// FieldCount returns the number of fields.
func (r Response) FieldCount() int {
	return len(r.Fields)
}

func boolToInt(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
