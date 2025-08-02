package xgm
import "github.com/mocham/xgw"
const IMTitle = "auto-sticky-im"

func InputMethodWidget(paster func(int)) {
    if xgw.ImWindow != 0 { xgw.RaiseWindow(xgw.ImWindow); return } // Map this window and put it to front, but don't focus on it.
    var picks []string
	strBuffer, pasteMode, winWidth, focusBackup := "", 0, xgw.Width*2/3, xgw.FocusWindow // Restore focus window later if xgw.ImWindow is focused
	renderCandidates := func(state *xgw.SingleRowState) {
		state.Instructions.PushBack("XPos#Save", " ")
		picks = PinyinCandidates(strBuffer) // This is based on the libgooglepinyin C library, interfaced using CGO.
		for id, pick := range picks {
			if state.Instructions.PushBack(xgw.FmtInt(id+1)) != nil { break }
			for _, pchar := range pick { if state.Instructions.PushBack(xgw.FmtChar(pchar)) != nil { break } }
		}
		state.Instructions.PushBack("XPos#Load")
	}
	keypress := func(detail byte, state *xgw.SingleRowState) int {
		if xgw.FocusWindow == xgw.ImWindow {
			xgw.FocusSet(focusBackup)
		} else {
			focusBackup = xgw.FocusWindow // The active window might have changed.
		}
		char := xgw.Conf.Keymap[detail]
		if len(char) == 0 { return 0 }
		if char == "=" { char = "+" }
		if ((char[0] >= 'a' && char[0] <= 'z') || char[0] == '+') && state.XPos + xgw.GlyphWidth < winWidth {
			if len(strBuffer) >= 20 { return 0 } //libgooglepinyin is bugged and will crash for long inputs
			state.Instructions.PushBack(char)
			strBuffer += char
			renderCandidates(state)
			state.Instructions.PushBack("Grab#Backspace")
			return 1
		}
		choice := int(char[0] - '1')
		if choice >= 0 && choice < len(picks) {
			xgw.SetClipboard("CLIPBOARD", picks[choice], BarXImage.Win) // Own clipboard and set selection text to picks[choice]
			state.Instructions.PushBack("Clear")
			strBuffer = ""
			paster(pasteMode)
			state.Instructions.PushBack("Ungrab#Backspace")
			return 1
		}
		switch char { // Handle special keys
		case "Backspace": 
			state.Instructions.PushBack(char)
			if len(strBuffer) >= 1 { strBuffer = strBuffer[:len(strBuffer) - 1] }
			if len(strBuffer) == 0 { 
				state.Instructions.PushBack("Ungrab#Backspace", "Clear")
			} else {
				renderCandidates(state)
			}
		case "-": pasteMode = 1 - pasteMode
		case "Escape":  return -1
        }
		return 1
    }
	defer func() { xgw.ImWindow = 0 } ()
	xgw.SingleRowGlyphWidget(IMTitle, 0, xgw.Height - xgw.GlyphHeight, winWidth, []uint16{0}, keypress, func (state *xgw.SingleRowState) { state.Instructions.PushBack("Grab#IM", "SetIM", "Raise") })
}

func PromptWidget(hint string, exec func(prompt string)) {
    if xgw.ImWindow != 0 { xgw.RaiseWindow(xgw.ImWindow); return }
    strBuffer, winWidth, focusBackup := "", xgw.Width*2/3, xgw.FocusWindow
    defer func() { xgw.ImWindow = 0 } ()
	xgw.SingleRowGlyphWidget(IMTitle, 0, xgw.Height - xgw.GlyphHeight, winWidth, []uint16{0, 1}, func(detail byte, state *xgw.SingleRowState) int {
		if xgw.FocusWindow == xgw.ImWindow { xgw.FocusSet(focusBackup) }
		char := xgw.Conf.Keymap[detail]
		if len(char) == 1 && state.XPos + xgw.GlyphWidth < winWidth {
			state.Instructions.PushBack(char)
			strBuffer += char
		} else {
			switch char { // Handle special keys
			case "Backspace": state.Instructions.PushBack(char); if len(strBuffer) >= 1 { strBuffer = strBuffer[:len(strBuffer) - 1] }
			case "Escape": return -1
			case "Return": exec(strBuffer); return -1
			}
		}
		return 1
	}, func (state *xgw.SingleRowState) {
		state.Instructions.PushBack("Grab#Return", "Grab#Backspace", "Grab#IM", "SetIM", "Raise")
		for _, char := range hint { state.Instructions.PushBack(xgw.FmtChar(char)) } // Draw hint string
	})
}
