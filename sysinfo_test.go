package sysinfo

import (
	"context"
	"testing"
)

func TestParseFree(t *testing.T) {
	out := `               total        used        free      shared  buff/cache   available
Mem:            30Gi       7.8Gi        17Gi       2.0Gi       7.9Gi        22Gi
Swap:          511Mi          0B       511Mi
`
	m, err := parseFree(out)
	if err != nil {
		t.Fatal(err)
	}
	want := Mem{Total: "30Gi", Used: "7.8Gi", Free: "17Gi", Shared: "2.0Gi", BuffCache: "7.9Gi", Available: "22Gi"}
	if *m != want {
		t.Fatalf("got %+v, want %+v", *m, want)
	}
}

func TestParseDf(t *testing.T) {
	out := `Filesystem      Size  Used Avail Capacity Mounted on
tmpfs           3.1G  2.5M  3.1G       1% /run
/dev/vda1        40G   21G   17G      56% /
/dev/sda1       916G  748G  122G      86% /media/data
`
	d, err := parseDf(out, "/dev/vda1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Filesystem != "/dev/vda1" || d.Size != "40G" || d.UsePercent != "56%" || d.MountedOn != "/" {
		t.Fatalf("unexpected %+v", *d)
	}

	// Нет устройства — берём корневой раздел.
	d, err = parseDf(out, "/dev/vdb1")
	if err != nil || d.Filesystem != "/dev/vda1" {
		t.Fatalf("fallback: %+v, %v", d, err)
	}
}

// TestCollect — вызывает реальные free/df на текущей машине.
func TestCollect(t *testing.T) {
	info := Collect(context.Background())
	if info.Mem == nil {
		t.Fatalf("mem: %s", info.MemErr)
	}
	if info.Disk == nil {
		t.Fatalf("disk: %s", info.DiskErr)
	}
	t.Logf("mem: %+v", *info.Mem)
	t.Logf("disk: %+v", *info.Disk)
}
