// +build openbsd
// +build !nomeminfo

package metric

import (
	"fmt"
)

/*
#include <sys/param.h>
#include <sys/types.h>
#include <sys/sysctl.h>

int
sysctl_uvmexp(struct uvmexp *uvmexp)
{
        static int uvmexp_mib[] = {CTL_VM, VM_UVMEXP};
        size_t sz = sizeof(struct uvmexp);

        if(sysctl(uvmexp_mib, 2, uvmexp, &sz, NULL, 0) < 0)
                return -1;

        return 0;
}

*/
import "C"

func (c *meminfoCollector) getMemInfo() (map[string]float64, error) {
	var uvmexp C.struct_uvmexp

	if _, err := C.sysctl_uvmexp(&uvmexp); err != nil {
		return nil, fmt.Errorf("sysctl CTL_VM VM_UVMEXP failed: %v", err)
	}

	ps := float64(uvmexp.pagesize)

	// see uvm(9)
	return map[string]float64{
		"active_bytes":                  ps * float64(uvmexp.active),
		"cache_bytes":                   ps * float64(uvmexp.vnodepages),
		"free_bytes":                    ps * float64(uvmexp.free),
		"inactive_bytes":                ps * float64(uvmexp.inactive),
		"size_bytes":                    ps * float64(uvmexp.npages),
		"swap_size_bytes":               ps * float64(uvmexp.swpages),
		"swap_used_bytes":               ps * float64(uvmexp.swpginuse),
		"swapped_in_pages_bytes_total":  ps * float64(uvmexp.pgswapin),
		"swapped_out_pages_bytes_total": ps * float64(uvmexp.pgswapout),
		"wired_bytes":                   ps * float64(uvmexp.wired),
		"MemAvailable_bytes":            ps * float64(uvmexp.free+uvmexp.vnodepages+uvmexp.inactive),
	}, nil
}
