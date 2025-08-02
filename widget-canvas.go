package xgm
import "github.com/mocham/xgw"
func MultipleCanvasWidget(width, height int, title string, images []string, getImageFromPath func(string, int, int) (xgw.RGBAData, error)) {
	if len(images) == 0 { return }
	cache, img_id, old_id, xOffset, numInput, cacheCount := map[int]xgw.RGBAData{}, 0, 0, 0, 0, 0
	draw := func (ximg *xgw.XImage, i int) (int, int) {
		if i < 0 || i >= len(images) { return 0, 0 }
		img, exists := cache[i]
		if !exists {
			if cacheCount += 1; cacheCount % 200 == 0 { cache = make(map[int]xgw.RGBAData) }
			var err error
			if img, err = getImageFromPath(images[i], width, height); err != nil { return 0, 0 } else { cache[i] = img }
		}
		if xOffset + img.Width > width { return 0, 0 }
		ximg.XDraw(img, xOffset, 0)
		if img.Height < height { ximg.XDraw(xgw.BlankImage(img.Width, height - img.Height), xOffset, img.Height) }
		return img.Width, img.Height
	}
	xgw.UniversalWidget(title, 0, 0, width, height, func(ximg *xgw.XImage) (xoff, max_ht int) {
		xOffset, max_ht, old_id = 0, 100, img_id
		for {
			if img_id < 0 { img_id += len(images); continue }
			if img_id >= len(images) { img_id -= len(images); continue }
			delta, ht := draw(ximg, img_id)
			if delta == 0 { break }
			img_id += 1
			xOffset += delta
			if ht > max_ht { max_ht = ht }
		}
		xoff = xOffset
		return
	}, nil, func (detail byte) int {
		switch detail {
		case 10, 11, 12, 13, 14, 15, 16, 17, 18, 19: numInput = numInput * 10 + int(detail + 1) % 10; return 0
		case 36: if numInput > 0 { img_id, numInput  = numInput, 0 } // Enter
		case 24: return -1 //"q"
		case 113: if len(images) == 0 { return 0 }; img_id = old_id - 1
		case 114: if len(images) == 0 { return 0 }
		default: return 0
		}
		return 1
	}, nil, xgw.WindowRaiseFocuser)
}

func ScrotWidget() {
	width, height := xgw.Width, xgw.Height
	_, data32 := xgw.Screenshot(0, 0, width, height)
	if data32 == nil { return }
	state, scrImg := 0, xgw.RGBAData{Pix: data32, Stride: width*4, Width: width, Height: height}
	var coord [4]int
	xgw.UniversalWidget("auto-scrot", 0, 0, width, height, func(ximg *xgw.XImage) (int, int) {
		switch state {
		case 0: ximg.XDraw(scrImg, 0, 0)
		case 1: ximg.XDraw(xgw.BlankImage(width, coord[1]), 0, 0); ximg.XDraw(xgw.BlankImage(coord[0], height), 0, 0)
		}
		return 0, 0
	}, func (detail byte, x, y int16) int {
		coord[state*2], coord[state*2+1] = int(x), int(y)
		state += 1
		if state < 2 { return 1 }
		w, h := coord[2] - coord[0], coord[3] - coord[1]
		output := make([]uint32, 0, w*h)[:w*h]
		for i:=coord[1]; i < coord[3]; i++ { copy(output[(i-coord[1])*w:(i-coord[1]+1)*w], data32[i*width+coord[0]:(i+1)*width+coord[0]]) }
		SavePNG(output, w, h, "/tmp/screenshot.png")
		return -1
	}, func (detail byte) int { return -1 }, nil, xgw.WindowRaiseFocuser)
}

