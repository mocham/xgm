package xgm
/*
#cgo LDFLAGS: -L/output/static-libs -l:libstb.a -l:libturbojpeg.a -l:libwebp.a -l:libgooglepinyin.a -l:libasound.a -l:libstdc++.a -static-libgcc -lm
#cgo CFLAGS: -I/output/include/
#include <libbar_pinyin.h>
#include "CPlugins/src/plugin-snd.c"
#include "CPlugins/src/plugin-img.c"
*/
import "C"
import (
	"os"
	"bytes"
	"path/filepath"
	"encoding/json"
	"io"
	"unsafe"
	"strings"
	"github.com/hajimehoshi/go-mp3"
	"github.com/mocham/xgw"
	_ "embed"
)
var (
	pynFlag bool
	//go:embed alsa.conf
	alsaConf []byte
	pinyinBuffer [1024]byte
)
type cStr struct { data []byte; Ptr *C.char }
func CStr(str string) (ret cStr) { ret.data = xgw.CStrBytes(str); ret.Ptr = xgw.Ptr[C.char](&ret.data[0]); return }

func init() {
    configJSON, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".local", "bar.json"))
    logAndExit(err, json.NewDecoder(bytes.NewReader(configJSON)).Decode(&Conf))
    BarXImage = xgw.NewXImage(xgw.Width-xgw.GlyphWidth*50, xgw.Height-xgw.GlyphHeight, xgw.GlyphWidth*50, xgw.GlyphHeight, xgw.BarTitle)
	InitBarData(BarXImage.Win, "init")
    if BarXImage == nil { logAndExit(xgw.ErrXImg) }
    for modkey, arr := range Conf.KeyBindings {
        numericKey := xgw.ParseInt(modkey)
        keyMap[numericKey] = arr
        BarXImage.Grab(uint16(numericKey/10000), byte(numericKey%10000))
    }
	C.init_pinyin(CStr(xgw.ExpandHome(Conf.PinyinDB)).Ptr, CStr(xgw.ExpandHome(Conf.PinyinUserDB)).Ptr); pynFlag = true
	logAndExit(os.WriteFile("/tmp/alsa.conf", alsaConf, 0644))
	os.Setenv("ALSA_CONFIG_PATH", "/tmp/alsa.conf")
	C.init_alsa() 
	go initMouseMonitor()
}

func Cleanup() {
	if mouseProcess != nil { mouseProcess.Kill(); mouseProcess = nil}
	if BarXImage != nil { BarXImage.Destroy() }
	if pynFlag { C.cleanup_pinyin() }
	xgw.Cleanup()
}

func SwitchAlsaMode() { C.switch_alsa_mode() }
func GetAlsaVolume() int { return int(C.get_alsa_volume()) }
func SetAlsaVolume(percentage int) int { return int(C.set_alsa_volume(C.int(percentage))) }

func PinyinCandidates(input string) []string {
	C.get_pinyin_candidates(CStr(input).Ptr, xgw.Ptr[C.char](&pinyinBuffer[0]), 1024)
	return strings.Split(C.GoString(xgw.Ptr[C.char](&pinyinBuffer[0])), "\n")
}

func PlayMP3(filename string) error {
	file, err := os.Open(filename)
	if err != nil { return err }
	defer file.Close()
	decoder, err := mp3.NewDecoder(file)
	if err != nil { return err }
	player := C.alsa_open(C.uint(decoder.SampleRate()), 2)
	if player == nil { return xgw.ErrLoad }
	defer C.alsa_close(player)
	buf := make([]byte, 0, 4096)[:4096]
	for {
		n, err := decoder.Read(buf)
		if err == io.EOF || n == 0 { break }
		if err != nil { return err }
		C.alsa_send(player, unsafe.Pointer(&buf[0]), C.size_t(n))
	}
	return nil
}

func DecodeImage(cData unsafe.Pointer, dataLen int, cOutData unsafe.Pointer, maxWidth, maxHeight int) (C.int, C.int, error) {
	if dataLen == 0 || dataLen < -1 || maxWidth <= 0 || maxHeight <= 0 || cOutData == nil { return  0, 0, xgw.ErrParam }
	var width, height, retState, outWidth, outHeight C.int
	var pixels *C.uchar
	var imgType C.alloc_type_t
	if dataLen == -1 { // If dataLen == -1, then cData is treated as a filepath, and is treated as the actual bytes for an image format if otherwise
		retState = C.img_load_bgra((*C.char)(cData), &pixels, &width, &height, &imgType, 0)
	} else {
		retState = C.img_load_bgra_from_memory((*C.uchar)(cData), C.int(dataLen), &pixels, &width, &height, &imgType, 0)
	}
	switch retState {
	case C.IMG_SUCCESS: defer C.img_free_buffer(unsafe.Pointer(pixels), imgType)
	case C.IMG_ERROR_LOAD: return 0, 0, xgw.ErrLoad
	case C.IMG_ERROR_INVALID_PARAM: return 0, 0, xgw.ErrParam
	default: return 0, 0, xgw.ErrUnknown
	}
	var flip C.int = 1
	if imgType == C.ALLOC_JPEG_TURBO { flip = 0 }
	switch C.img_resize_bgra_to_fit(pixels, width, height, C.int(maxWidth), C.int(maxHeight), (*C.uchar)(cOutData), &outWidth, &outHeight, flip) {
	case C.IMG_SUCCESS: return C.int(outWidth), C.int(outHeight), nil
	case C.IMG_ERROR_RESIZE: return 0, 0, xgw.ErrResize
	case C.IMG_ERROR_INVALID_PARAM: return 0, 0, xgw.ErrParam
	default: return 0, 0, xgw.ErrUnknown
	}
}

func MallocAndDecodeImageFromFile(filename string) (xgw.RGBAData, error) {
    var width, height C.int
    var pixels *C.uchar
    var imgType C.alloc_type_t
	switch C.img_load_bgra(CStr(filename).Ptr, xgw.Ptr[*C.uchar](&pixels), &width, &height, &imgType, 1) {
	case C.IMG_SUCCESS: defer C.img_free_buffer(unsafe.Pointer(pixels), imgType)
	case C.IMG_ERROR_LOAD: return xgw.RGBAData{}, xgw.ErrLoad
	case C.IMG_ERROR_INVALID_PARAM: return xgw.RGBAData{}, xgw.ErrParam
	default: return xgw.RGBAData{}, xgw.ErrUnknown
    }
	length := int(width*height)
	ret := xgw.RGBAData{Pix: make([]uint32,0,length)[:length], Width:int(width), Height:int(height), Stride: 4*int(width)}
	copy(ret.Pix, xgw.Array[uint32](pixels, length))
    return ret, nil
}

func SavePNG(data []uint32, width, height int, filename string) { C.save_png(xgw.Ptr[C.uchar](&data[0]), C.int(width), C.int(height), CStr(filename).Ptr) }

func EncodeWebp(data []uint32, width, height int, quality float32) (ret []byte) {
	var size C.size_t
	output := C.encode_rgba_to_webp(xgw.Ptr[C.uint8_t](&data[0]), C.int(width), C.int(height), C.int(width*4), C.float(quality), &size)
	if output == nil { return nil }
	defer C.free(unsafe.Pointer(output))
	ret = make([]byte, 0, int(size))[:int(size)]
	copy(ret, xgw.Array[byte](output, int(size)))
	return
}
