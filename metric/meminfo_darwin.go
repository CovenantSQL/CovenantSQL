// +build !nomeminfo

package metric

// #include <mach/mach_host.h>.
import "C"

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (c *meminfoCollector) getMemInfo() (map[string]float64, error) {
	infoCount := C.mach_msg_type_number_t(C.HOST_VM_INFO_COUNT)
	vmstat := C.vm_statistics_data_t{}
	ret := C.host_statistics(
		C.host_t(C.mach_host_self()),
		C.HOST_VM_INFO,
		C.host_info_t(unsafe.Pointer(&vmstat)),
		&infoCount,
	)
	if ret != C.KERN_SUCCESS {
		return nil, fmt.Errorf("couldn't get memory statistics, host_statistics returned %d", ret)
	}
	totalb, err := unix.Sysctl("hw.memsize")
	if err != nil {
		return nil, err
	}
	// Syscall removes terminating NUL which we need to cast to uint64
	total := binary.LittleEndian.Uint64([]byte(totalb + "\x00"))

	ps := C.natural_t(syscall.Getpagesize())
	return map[string]float64{
		"active_bytes_total":      float64(ps * vmstat.active_count),
		"inactive_bytes_total":    float64(ps * vmstat.inactive_count),
		"wired_bytes_total":       float64(ps * vmstat.wire_count),
		"free_bytes_total":        float64(ps * vmstat.free_count),
		"swapped_in_pages_total":  float64(ps * vmstat.pageins),
		"swapped_out_pages_total": float64(ps * vmstat.pageouts),
		"bytes_total":             float64(total),
		"MemAvailable_bytes":      float64(ps * (vmstat.free_count + vmstat.inactive_count)),
	}, nil
}
