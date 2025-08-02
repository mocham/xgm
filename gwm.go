package main
import (
	"sync/atomic"
	"strings"
	"slices"
	"github.com/mocham/xgw"
	"sort"
)
const tilePref, hijackedPref, stickyPref = "tile-", "auto-", "auto-sticky"
var deskLock uint32
func setBarData(win Window, data string) {
    xgw.WinStates[win] = xgw.WindowState{Mapped: xgw.WinStates[win].Mapped, BarData: data}
	xgw.SendString(win, xgw.AtomMap[xgw.Conf.BarAtom], data)
}

func extractDesktopName(s string) (string, string) {
    idx := strings.LastIndex(s, "@")
    if idx == -1 { return s, "" }
    if idx == 0 { return "", s[1:] }
    if idx + 1 == len(s) { return s[:idx-1], ""}
	if strings.HasSuffix(s, "_RAW") { return s[:idx-1], "_RAW" }
    return s[:idx - 1], s[idx + 1:]
}

func adjacentDesktop(delta int) {
	if xgw.DeskID += delta; xgw.DeskID >= len(desktops) { xgw.DeskID = 0 }
	if xgw.DeskID < 0 { xgw.DeskID = len(desktops) - 1 }
    if desktops[xgw.DeskID] == "vm" { xgw.DeskID = 0 }
    showDesktop(desktops[xgw.DeskID])
}

func floatWindow(win Window, unfloat bool) {
	_, _, w, h := xgw.GetGeometry(win)
	xgw.ResizeWindow(win, (xgw.Width-w)/2, (xgw.Height-h)/2, w, h)
	if barData := xgw.WinStates[win].BarData; !strings.HasPrefix(barData, hijackedPref) {
		setBarData(win, hijackedPref + barData)
	} else if unfloat && !strings.HasPrefix(barData, stickyPref)  {
		setBarData(win, tilePref + barData[len(hijackedPref):])
		tileWindows(0)
	}
}

func updateDesktop(resetCurrent bool) {
    if atomic.LoadUint32(&deskLock) == 1 { return }
	atomic.StoreUint32(&deskLock, 1)
	defer atomic.StoreUint32(&deskLock, 0)
	keys := make(map[string]bool)
	ucfg.hasVM, xgw.DesktopWins = false, xgw.DesktopWins[:0]
    for win, state := range xgw.WinStates {
		if win == xgw.BarWindow { continue }
		_, desk := extractDesktopName(state.BarData)
		if desk == "vm" { ucfg.hasVM = true }
		if desk == "_RAW" || desk == "" { continue }
		if desk == xgw.CurrentDesktop { xgw.DesktopWins = append(xgw.DesktopWins, win) }
		keys[desk] = true
		if state.Mapped && resetCurrent { xgw.CurrentDesktop = desk }
    }
	if xgw.CurrentDesktop == "" { xgw.CurrentDesktop = "1" }
    keys[xgw.CurrentDesktop] = true
    desktops = make([]string, 0, len(keys))
	for name, _ := range keys { desktops = append(desktops, name) }
	sort.Strings(desktops)
	for id, name := range desktops { if name == xgw.CurrentDesktop { xgw.DeskID = id } }
	drawDesktopInfo()
}

func nextFocus(update bool) {
	if update { updateDesktop(true) }
	visited, joined := false, slices.Concat(xgw.DesktopWins, xgw.StickyWins)
    if len(joined) == 0 { return }
	sort.Slice(joined, func(i, j int) bool { return joined[i] < joined[j] })
	for _, win := range joined {
		if visited { xgw.FocusSet(win); return }
		if win == xgw.FocusWindow { visited = true }
	}
	xgw.FocusSet(joined[0])
}

func tileWindows(extraWin Window) {
	cands, top, fId := []Window{}, 0, 0
	if updateDesktop(true); ucfg.positionFlag != 0 { top = xgw.GlyphHeight }
    for _, win := range xgw.DesktopWins { if barData := xgw.WinStates[win].BarData; win != xgw.BarWindow && win != extraWin && xgw.WinStates[win].Mapped && !strings.Contains(barData, hijackedPref) && strings.Contains(barData, tilePref) { cands = append(cands, win) } }
	if extraWin != 0 { cands = append(cands, extraWin) }
	sort.Slice(cands, func(i, j int) bool { return cands[i] < cands[j] })
    if len(cands) == 0 { return }
	if len(cands) != 2 {
		eachWidth := (xgw.Width - 20*len(cands) + 20) / len(cands)
		if eachWidth < 1 { eachWidth = 1 }
		for i, win := range cands {
			h := xgw.Height
			if (i+1)*(eachWidth+20) > xgw.Width*3/5 { h -= xgw.GlyphHeight }
			xgw.ResizeWindow(win, i*(eachWidth+20), top, eachWidth, h)
		}
		return
	}
	if cands[1] == xgw.FocusWindow { fId = 1 }
	x1, y1, w1, h1, x2, y2, w2, h2 := 0, top, xgw.Width/2-10, xgw.Height, xgw.Width/2+10, top, xgw.Width/2-10, xgw.Height-xgw.GlyphHeight
	if x, _, w, _ := xgw.GetGeometry(cands[fId]); w*3>=xgw.Width && w*3<=xgw.Width*2 {
		if 2*x>=xgw.Width || xgw.Width<=2*x+w {
			x1, x2, w1, w2, h1, h2 = x, 0, xgw.Width-x, x-20, h2, h1
		} else {
			x2, w1, w2 = x+w+20, x+w, xgw.Width-x-w-20
		}
	}
	xgw.ResizeWindow(cands[fId], x1, y1, w1, h1)
	xgw.ResizeWindow(cands[1-fId], x2, y2, w2, h2)
}

func windowMap(win Window) {
	barData := xgw.WinStates[win].BarData
	xgw.DesktopWins, xgw.WinStates[win] = append(xgw.DesktopWins, win), xgw.WindowState{Mapped: true, BarData: barData}
	arr := strings.Split(barData, "@")
	if len(arr) <= 4 { return }
	if arr[1] == "" { if arr[1] = xgw.GetTitle(win); arr[1] != "" { initBarData(win, xgw.CurrentDesktop) } }
	if barData[len(barData) - 4:] == "_RAW" { setBarData(win, barData[:len(barData) - 4]) }
	switch {
	case strings.Contains(barData, stickyPref): xgw.StickyWins = append(xgw.StickyWins, win); xgw.FocusSet(win)
	case strings.Contains(barData, hijackedPref): if x, y, w, h := getWindowUserConfig(arr[1]); w > 0 { xgw.ResizeWindow(win, x, y, w, h) } else if w == 0 { floatWindow(win, false) }; xgw.FocusSet(win)
	case strings.Contains(barData, tilePref): xgw.FocusSet(win)
	}
}

func initBarData(win Window, cDesk string) (tiling bool) {
	title, classParts := xgw.GetTitle(win), strings.Split(string(xgw.QueryBytes(win, "WM_CLASS")), "\x00")
    prefix, suffix, str := "normal", cDesk, strings.Join([]string{title, classParts[0], classParts[len(classParts) - 1]}, "@")
    switch {
    case strings.Contains(str, stickyPref): prefix = stickyPref
    case strings.Contains(str, hijackedPref): prefix = hijackedPref
    case slices.Contains(ucfg.ForcedTilingClasses, classParts[0]) || strings.Contains(str, tilePref): prefix = tilePref; tiling = true
    }
    setBarData(win, prefix + "@" + str + "@" + suffix)
	return
}

func moveWindowToDesktop(suffix string, winStr ...string) {
	win := xgw.FocusWindow
	if len(winStr) > 0 { win = Window(xgw.ParseInt(winStr[0])) }
    if win == 0 || win == xgw.BarWindow || win == xgw.Root { return }
	if suffix != xgw.CurrentDesktop { xgw.Unmap(win) } else { xgw.Map(win) }
	barType, _ := extractDesktopName(xgw.WinStates[win].BarData)
	setBarData(win, barType + "@" + suffix)
}

func showDesktop(suffix string) {
    if len(suffix) == 0 { return }
    xgw.CurrentDesktop = suffix 
	updateDesktop(false)
    unmaps, hasFocus := []Window{}, false
    for win, state := range xgw.WinStates { if state.Mapped && !strings.HasSuffix(state.BarData, suffix) && !strings.HasPrefix(state.BarData, stickyPref) { unmaps = append(unmaps, win) } }
	raiser := func (win Window) { if !xgw.WinStates[win].Mapped && xgw.Map(win) && !hasFocus { xgw.FocusSet(win); hasFocus = true } }
	for _, win := range xgw.DesktopWins { raiser(win) }
	for _, win := range xgw.StickyWins { raiser(win) }
    for _, win := range unmaps { xgw.Unmap(win) }
}

func windowUnmap(win Window) {
	barData := xgw.WinStates[win].BarData
	xgw.WinStates[win] = xgw.WindowState{Mapped: false, BarData: barData}
	if strings.HasSuffix(barData, "@" + xgw.CurrentDesktop) && !strings.HasSuffix(barData, "_RAW") { setBarData(win, barData + "_RAW") }
	if strings.Contains(barData, stickyPref) { 
		xgw.StickyWins = xgw.RemoveElement(xgw.StickyWins, win) 
	} else {
		xgw.DesktopWins = xgw.RemoveElement(xgw.DesktopWins, win) 
	}
}

func desktopEventLoop() {
    for {
        ev, err := barXImage.Conn.WaitForEvent()
        if err != nil || ev == nil { continue }
        switch event := ev.(type) {
        case xgw.EXMap: windowMap(event.Window)
        case xgw.EXCreate: if initBarData(event.Window, xgw.CurrentDesktop + "_RAW") {tileWindows(event.Window)} 
        case xgw.EXDestroy: updateDesktop(true); delete(xgw.WinStates, event.Window)
        case xgw.EXUnmap: windowUnmap(event.Window)
        case xgw.EXKey: if action, exists := ucfg.keyMap[int(event.State)*10000+int(event.Detail)]; exists { doAction(action...) }; xgw.SetXTime(uint32(event.Time))
        case xgw.EXSel: if event.Owner == xgw.BarWindow { xgw.UseClipboard(event.Requestor, event.Property, event.Target, event.Selection, event.Time) }
        }
    }
}
