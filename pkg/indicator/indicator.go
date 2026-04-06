package indicator

/*
#cgo pkg-config: ayatana-appindicator3-0.1 gtk+-3.0
#include <libayatana-appindicator/app-indicator.h>
*/
import "C"
import "unsafe"

// Indicator wraps a libayatana-appindicator AppIndicator.
type Indicator struct {
	native *C.AppIndicator
}

// New creates an AppIndicator. iconName is the filename stem (no extension).
// iconThemePath is the directory containing the SVG/PNG files.
func New(id, iconName, iconThemePath string) *Indicator {
	cID := C.CString(id)
	defer C.free(unsafe.Pointer(cID))
	cIcon := C.CString(iconName)
	defer C.free(unsafe.Pointer(cIcon))
	cPath := C.CString(iconThemePath)
	defer C.free(unsafe.Pointer(cPath))

	native := C.app_indicator_new_with_path(
		cID, cIcon,
		C.APP_INDICATOR_CATEGORY_APPLICATION_STATUS,
		cPath,
	)
	C.app_indicator_set_status(native, C.APP_INDICATOR_STATUS_ACTIVE)
	return &Indicator{native: native}
}

// SetIcon changes the displayed icon by name (stem only, no extension).
func (ind *Indicator) SetIcon(iconName string) {
	cName := C.CString(iconName)
	defer C.free(unsafe.Pointer(cName))
	C.app_indicator_set_icon_full(ind.native, cName, cName)
}

// SetMenu attaches a GtkMenu to the indicator.
// menuPtr should be obtained from gotk3's menu.Native().
func (ind *Indicator) SetMenu(menuPtr uintptr) {
	ptr := unsafe.Pointer(menuPtr) //nolint:govet // necessary for CGO
	C.app_indicator_set_menu(ind.native, (*C.GtkMenu)(ptr))
}
