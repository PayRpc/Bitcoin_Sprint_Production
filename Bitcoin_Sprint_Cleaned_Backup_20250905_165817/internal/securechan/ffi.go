//go:build cgo
// +build cgo

package securechan

/*
#cgo LDFLAGS: -L. -lsecurechannel
#include <stdlib.h>
#include <stdint.h>
extern void* secure_channel_new(const char* endpoint);
extern void secure_channel_free(void* pool);
extern bool secure_channel_start();
extern bool secure_channel_stop();
*/
import "C"
import "unsafe"

type Channel struct{ ptr unsafe.Pointer }

func New(endpoint string) *Channel {
	cstr := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cstr))
	ptr := C.secure_channel_new(cstr)
	return &Channel{ptr: ptr}
}

func (c *Channel) Start() bool { return bool(C.secure_channel_start()) }
func (c *Channel) Stop() bool  { return bool(C.secure_channel_stop()) }
func (c *Channel) Free()       { C.secure_channel_free(c.ptr) }
