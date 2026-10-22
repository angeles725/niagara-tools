package hostbudget

import "syscall"

// statfs is the host Statfs that Main uses; command tests replace it with a fixed Disk.
var statfs = realStatfs

// realStatfs reads one filesystem with statfs(2), counting as df does.
func realStatfs(path string) (Disk, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Disk{}, err
	}
	bs := uint64(st.Bsize)
	return Disk{Used: (st.Blocks - st.Bfree) * bs, Avail: st.Bavail * bs}, nil
}
