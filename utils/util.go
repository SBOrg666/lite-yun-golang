package utils

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"strconv"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/disk"
	"time"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"bytes"
	"os"
	"strings"
	"path/filepath"
	"golang.org/x/sys/unix"
	"syscall"
	"os/user"
	"archive/zip"
	"io"
)

type DirItem struct {
	Url        string
	DirName    string
	Permission string
	Size       string
	Owner      string
	Group      string
	Mtime      string
	Access     bool
}

type FileItem struct {
	FileName   string
	Permission string
	Size       string
	Owner      string
	Group      string
	Mtime      string
	Access     bool
}

const defaultBufSize = 4096

func tail(filename string, n int) (lines []string, err error) {
	f, e := os.Stat(filename)
	if e == nil {
		size := f.Size()
		var fi *os.File
		fi, err = os.Open(filename)
		if err == nil {
			b := make([]byte, defaultBufSize)
			sz := int64(defaultBufSize)
			nn := n
			bTail := bytes.NewBuffer([]byte{})
			istart := size
			flag := true
			for flag {
				if istart < defaultBufSize {
					sz = istart
					istart = 0
					//flag = false
				} else {
					istart -= sz
				}
				_, err = fi.Seek(istart, os.SEEK_SET)
				if err == nil {
					mm, e := fi.Read(b)
					if e == nil && mm > 0 {
						j := mm
						for i := mm - 1; i >= 0; i-- {
							if b[i] == '\n' {
								bLine := bytes.NewBuffer([]byte{})
								bLine.Write(b[i+1 : j])
								j = i
								if bTail.Len() > 0 {
									bLine.Write(bTail.Bytes())
									bTail.Reset()
								}

								if (nn == n && bLine.Len() > 0) || nn < n { //skip last "\n"
									lines = append(lines, bLine.String())
									nn --
								}
								if nn == 0 {
									flag = false
									break
								}
							}
						}
						if flag && j > 0 {
							if istart == 0 {
								bLine := bytes.NewBuffer([]byte{})
								bLine.Write(b[:j])
								if bTail.Len() > 0 {
									bLine.Write(bTail.Bytes())
									bTail.Reset()
								}
								lines = append(lines, bLine.String())
								flag = false
							} else {
								bb := make([]byte, bTail.Len())
								copy(bb, bTail.Bytes())
								bTail.Reset()
								bTail.Write(b[:j])
								bTail.Write(bb)
							}
						}
					}
				}
			}
			//func (f *File) Seek(offset int64, whence int) (ret int64, err error)
			//func (f *File) Read(b []byte) (n int, err error) {
		}
		defer fi.Close()
	}
	return
}

func GetLog_Info(fname string) string {
	lns, err := tail(fname, 25)
	if err != nil {
		return "Could not find " + fname
	}
	s := make([]string, 0)
	for _, v := range lns {
		s = append(s, v)
	}
	return fmt.Sprint(strings.Join(s, "\r\n"))
}

func GetCpu_Info() []float64 {
	cpu_Percent, err := cpu.Percent(0, true)
	checkErr(err)
	return cpu_Percent
}

func GetSys_Info() []string {
	sys_Info, err := host.Info()
	checkErr(err)
	cpuInfo, err := cpu.Info()
	checkErr(err)
	virtualMemory, err := mem.VirtualMemory()
	checkErr(err)
	bootTime, err := host.BootTime()
	checkErr(err)
	return []string{sys_Info.Hostname,
		sys_Info.Platform,
		sys_Info.KernelVersion,
		cpuInfo[0].ModelName,
		strconv.FormatFloat(cpuInfo[0].Mhz, 'f', 1, 64) + " Mhz",
		fmt.Sprint(virtualMemory.Total),
		fmt.Sprint(time.Unix(int64(bootTime), 0)),
	}

}

func GetMem_Info() []string {
	virtualMemory, err := mem.VirtualMemory()
	checkErr(err)
	return []string{fmt.Sprint(virtualMemory.Used), fmt.Sprint(virtualMemory.Total)}

}

func GetSwap_Info() []string {
	swapMemory, err := mem.SwapMemory()
	checkErr(err)
	return []string{fmt.Sprint(swapMemory.Used), fmt.Sprint(swapMemory.Total)}

}

func GetDisk_Info() [][]string {
	partitions, err := disk.Partitions(false)
	checkErr(err)
	mountpoints := make([]string, 0)
	for _, partition := range partitions {
		mountpoints = append(mountpoints, partition.Mountpoint)
	}
	usageStats := make([]disk.UsageStat, 0)
	for _, moutpoint := range mountpoints {
		stat, err := disk.Usage(moutpoint)
		checkErr(err)
		usageStats = append(usageStats, *stat)
	}
	used := make([]string, 0)
	for _, u := range usageStats {
		used = append(used, fmt.Sprint(u.Used))
	}
	total := make([]string, 0)
	for _, t := range usageStats {
		total = append(total, fmt.Sprint(t.Total))
	}
	disk_info := make([][]string, 0)
	for i, mountpoint := range mountpoints {
		info := make([]string, 0)
		info = append(info, mountpoint, used[i], total[i])
		disk_info = append(disk_info, info)
	}
	return disk_info
}

func GetNetwork_Info() [][]string {
	m := make([]int, 0)
	month := time.Now().Month()
	for range []int{0, 1, 2, 3, 4, 5} {
		if month > 0 {
			m = append(m, int(month))
		} else {
			m = append(m, int(month)+12)
		}
		month--
	}
	for i, j := 0, len(m)-1; i < j; i, j = i+1, j-1 {
		m[i], m[j] = m[j], m[i]
	}

	counter, err := net.IOCounters(false)
	checkErr(err)
	updata := make([]uint64, len(Upload_data))
	copy(updata, Upload_data)
	updata = append(updata, counter[0].BytesSent-InitUpload)
	downdata := make([]uint64, len(Download_data))
	copy(downdata, Download_data)
	downdata = append(downdata, counter[0].BytesRecv-InitDownload)

	mo := make([]string, 6)
	up := make([]string, 6)
	down := make([]string, 6)
	total := make([]string, 6)
	for i, _ := range []int{0, 1, 2, 3, 4, 5} {
		mo[i] = fmt.Sprint(m[i])
		up[i] = fmt.Sprint(updata[i])
		down[i] = fmt.Sprint(downdata[i])
		total[i] = fmt.Sprint(updata[i] + downdata[i])
	}

	return [][]string{mo, up, down, total}
}

func UpdateNetworkData() {
	if Current_Month != int(time.Now().Month()) {
		Current_Month = int(time.Now().Month())
		counter, err := net.IOCounters(false)
		checkErr(err)
		Upload_data = append(Upload_data, counter[0].BytesSent-InitUpload)
		Download_data = append(Download_data, counter[0].BytesRecv-InitDownload)
		Upload_data = Upload_data[1:]
		Download_data = Download_data[1:]
		InitUpload = counter[0].BytesSent
		InitDownload = counter[0].BytesRecv
	}
}

func convertStatus(s string) string {
	switch s {
	case "R":
		return "Running"
	case "S":
		return "Sleep"
	case "T":
		return "Stop"
	case "I":
		return "Idle"
	case "Z":
		return "Zombie"
	case "W":
		return "Wait"
	case "L":
		return "Lock"
	default:
		return ""
	}
}

func GetProcess_Info() ([]map[string]string, error) {
	res := make([]map[string]string, 0)
	processList, err := process.Processes()

	checkErr(err)
	for _, pro := range processList {
		pmap := make(map[string]string)
		pmap["pid"] = fmt.Sprint(pro.Pid)
		name, err := pro.Name()
		if err != nil {
			return nil, err
		}
		pmap["name"] = name
		username, err := pro.Username()
		if err != nil {
			return nil, err
		}
		pmap["username"] = username
		exe, err := pro.Exe()
		if err != nil {
			pmap["exe"] = ""
		} else {
			pmap["exe"] = exe
		}
		cpu_percent, err := pro.CPUPercent()
		if err != nil {
			return nil, err
		}
		pmap["cpu_percent"] = fmt.Sprint(cpu_percent)
		memory_percent, err := pro.MemoryPercent()
		if err != nil {
			return nil, err
		}
		pmap["memory_percent"] = fmt.Sprint(memory_percent)
		create_time, err := pro.CreateTime()
		if err != nil {
			return nil, err
		}
		pmap["create_time"] = fmt.Sprint(create_time)
		status, err := pro.Status()
		if err != nil {
			return nil, err
		}
		pmap["status"] = convertStatus(status)
		res = append(res, pmap)
	}
	return res, nil
}

func GetDirs(path string, allfiles []os.FileInfo) []DirItem {
	dirs := make([]DirItem, 0)
	for _, file := range allfiles {
		if file.IsDir() {
			var dir DirItem
			dir.DirName = file.Name()
			dir.Size = fmt.Sprint(file.Size())
			dir.Mtime = fmt.Sprint(file.ModTime())
			dir.Url = "/path?path=" + filepath.Join(path, file.Name())
			dir.Access = unix.Access(filepath.Join(path, file.Name()), unix.R_OK) == nil && unix.Access(filepath.Join(path, file.Name()), unix.X_OK) == nil
			dir.Permission = fmt.Sprint(file.Mode())
			u, err := user.LookupId(fmt.Sprint(file.Sys().(*syscall.Stat_t).Uid))
			if err != nil {
				dir.Owner = "unknown"
			} else {
				dir.Owner = u.Username
			}
			g, err := user.LookupGroupId(fmt.Sprint(file.Sys().(*syscall.Stat_t).Gid))
			if err != nil {
				dir.Group = "unknown"
			} else {
				dir.Group = g.Name
			}
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func GetFiles(path string, allfiles []os.FileInfo) []FileItem {
	files := make([]FileItem, 0)
	for _, file := range allfiles {
		if !file.IsDir() {
			var f FileItem
			f.FileName = file.Name()
			f.Size = fmt.Sprint(file.Size())
			f.Mtime = fmt.Sprint(file.ModTime())
			f.Access = unix.Access(filepath.Join(path, file.Name()), unix.R_OK) == nil
			f.Permission = fmt.Sprint(file.Mode())
			u, err := user.LookupId(fmt.Sprint(file.Sys().(*syscall.Stat_t).Uid))
			if err != nil {
				f.Owner = "unknown"
			} else {
				f.Owner = u.Username
			}
			g, err := user.LookupGroupId(fmt.Sprint(file.Sys().(*syscall.Stat_t).Gid))
			if err != nil {
				f.Group = "unknown"
			} else {
				f.Group = g.Name
			}
			files = append(files, f)
		}
	}
	return files
}

func Compress(files []*os.File, dest string) error {
	d, _ := os.Create(dest)
	defer d.Close()
	w := zip.NewWriter(d)
	defer w.Close()
	for _, file := range files {
		err := compress(file, "", w)
		if err != nil {
			return err
		}
	}
	return nil
}

func compress(file *os.File, prefix string, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		header, err := zip.FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name + "/"
		if err != nil {
			return err
		}
		_, err = zw.CreateHeader(header)
		if err != nil {
			return err
		}

		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
