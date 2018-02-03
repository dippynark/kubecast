package main

import (
	"bytes"
	"flag"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"unsafe"
	"net/http"
	"C"

	bpf "github.com/iovisor/gobpf/bcc"
	"github.com/golang/glog"
)

const (
	bufferSize = 256
	sessionIDHTTPHeader = "X-Session-ID"
	defaultServerAddress      = "localhost"
	defaultPort         = 5050
)

const source string = `
#include <uapi/linux/ptrace.h>

#include <linux/sched.h>
#include <linux/fs.h>
#include <linux/nsproxy.h>
#include <linux/ns_common.h>

// define data structures
struct sid_t {
    int sid;
};

#define BUFSIZE 256
struct tty_write_t {
    int count;
    char buf[BUFSIZE];
    unsigned int sessionid;
};

// define maps
BPF_HASH(active_sids, struct sid_t, u64);
BPF_PERF_OUTPUT(tty_writes);

// save_sid saves a sessionid generated from a call
// to setsid to the active_sids map
int save_sid(struct pt_regs *ctx) {

    struct sid_t sid_struct = {};
    pid_t sid = PT_REGS_RC(ctx);
    u64 time_ns = bpf_ktime_get_ns();

    sid_struct.sid = sid;

    active_sids.update(&sid_struct, &time_ns);

    return 0;

}

int kprobe__tty_write(struct pt_regs *ctx, struct file *file,
    const char __user *buf, size_t count)
{
    struct task_struct *task;
    struct pid_link pid_link;
    struct pid pid;
    int sessionid;

    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&pid_link, sizeof(pid_link), (void *)&task->group_leader->pids[PIDTYPE_SID]);
    bpf_probe_read(&pid, sizeof(pid), (void *)pid_link.pid);
		//sessionid = pid.numbers[0].nr;
		sessionid = 3;

    // build session struct key
    struct sid_t sid_key;
		sid_key.sid = sessionid;
		
		// if sid does not exist in our map then return
		//u64 *time_ns = active_sids.lookup(&sid_key);
    //if (!time_ns) {
    //    return 0;
    //}

    // bpf_probe_read() can only use a fixed size, so truncate to count
    // in user space:
    struct tty_write_t tty_write = {};
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)buf);
    if (count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = count;
    }

    // add sessionid to tty_write structure and submit
		tty_write.sessionid = sessionid;
    tty_writes.perf_submit(ctx, &tty_write, sizeof(tty_write));

    return 0;
}
`

type ttyWrite struct {
	Count int32
	Buf [bufferSize]byte
	SessionID int32
}

func main() {

	server := flag.String("server", defaultServerAddress, "address to connect to")
	port := flag.Int("port", defaultPort, "port to connect to")
	flag.Parse()

	address := fmt.Sprintf("%s:%d", *server, *port)
	//fmt.Printf("%s", address)

	m := bpf.NewModule(source, []string{})
	defer m.Close()

	ttyWriteKprobe, err := m.LoadKprobe("kprobe__tty_write")
	if err != nil {
		glog.Fatalf("Failed to load kprobe__tty_write: %s", err)
	}

	err = m.AttachKprobe("tty_write", ttyWriteKprobe)
	if err != nil {
		glog.Fatalf("Failed to attach kprobe__tty_write: %s", err)
	}

	setsidUretprobe, err := m.LoadUprobe("save_sid")
	if err != nil {
		glog.Fatalf("Failed to load save_sid: %s", err)
	}

	err = m.AttachUretprobe("c", "setsid", setsidUretprobe, -1)
	if err != nil {
		glog.Fatalf("Failed to attach save_sid: %s", err)
	}

	table := bpf.NewTable(m.TableId("tty_writes"), m)

	channel := make(chan []byte)

	perfMap, err := bpf.InitPerfMap(table, channel)
	if err != nil {
		glog.Fatalf("Failed to init perf map: %s", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	go func() {
		var event ttyWrite
		for {
			data := <-channel
			err := binary.Read(bytes.NewBuffer(data), binary.LittleEndian, &event)
			if err != nil {
				glog.Errorf("Failed to decode received data: %s", err)
				continue
			}
			buf := C.GoString((*C.char)(unsafe.Pointer(&event.Buf)))[0:event.Count]
			//fmt.Printf("%s", buf[0:event.Count])

			err = upload(int(event.SessionID), buf, address)
			if err != nil {
				glog.Errorf("Failed to upload buffer: %s", err)
				continue
			}

			glog.Infof("Successfully uploaded buffer with %d bytes", event.Count)
		}
	}()

	perfMap.Start()
	<-sig
	perfMap.Stop()
}

func upload(sid int, buf string, address string) error {

	b := bytes.NewBufferString(buf)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/upload", address), b)
	if err != nil {
		return err
	}

	req.Header.Add(sessionIDHTTPHeader, fmt.Sprintf("%d", sid))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}