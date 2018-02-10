#include <linux/kconfig.h>

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wgnu-variable-sized-type-not-at-end"
#pragma clang diagnostic ignored "-Waddress-of-packed-member"
#include <linux/ptrace.h>
#pragma clang diagnostic pop
#include <linux/version.h>
#include <linux/bpf.h>
#include "bpf_helpers.h"

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wtautological-compare"
#pragma clang diagnostic ignored "-Wgnu-variable-sized-type-not-at-end"
#pragma clang diagnostic ignored "-Wenum-conversion"
#include <net/sock.h>
#pragma clang diagnostic pop
#include <net/inet_sock.h>
#include <net/net_namespace.h>

// define structures
#define BUFSIZE 256
struct tty_write_t {
    int count;
    char buf[BUFSIZE];
    unsigned int sessionid;
};

struct sid_t {
    int sid; 
};

// define maps
struct bpf_map_def SEC("maps/active_sids") active_sids = {
	.type = BPF_MAP_TYPE_HASH,
	.key_size = sizeof(struct sid_t),
	.value_size = sizeof(uint64_t),
	.max_entries = 1024,
	.pinning = 0,
	.namespace = "",
};

struct bpf_map_def SEC("maps/tty_writes") tty_writes = {
	.type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
	.key_size = sizeof(int),
	.value_size = sizeof(u32),
	.max_entries = 2,
};

/*
// save_sid saves a sessionid generated from a call
// to setsid to the active_sids map
int save_sid(struct pt_regs *ctx) {

    struct sid_t sid_struct = {};
    int sid = PT_REGS_RC(ctx);
    uint64_t time_ns = bpf_ktime_get_ns();

    sid_struct.sid = sid;
    
    bpf_map_update(&sid_struct, &time_ns);

    return 0;

}
*/

SEC("kprobe/tty_write")
int kprobe__tty_write(struct pt_regs *ctx, struct file *file, const char __user *buf, size_t count)
{
    struct task_struct *task;
    struct pid_link pid_link;
    struct pid pid;
    int sessionid; 
    
    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    struct task_struct *group_leader;
    bpf_probe_read(&group_leader, sizeof(group_leader), (void *)&task->group_leader);
    bpf_probe_read(&pid_link, sizeof(pid_link), (void *)&group_leader->pids[PIDTYPE_SID]);    
    bpf_probe_read(&pid, sizeof(pid), (void *)pid_link.pid);
    sessionid = pid.numbers[0].nr;
   
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
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)buf); //(void *)buf);
    if (count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = count;
    }
    
    // add sessionid to tty_write structure and submit
    tty_write.sessionid = sessionid;
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));
    
    return 0;
}

/*
char buffer[BUFSIZE];
char *test = (char *)buf;
bpf_probe_read(buffer, sizeof(buffer), (void *)ctx->di);
SEC("kprobe/sys_open")
void bpf_sys_open(struct pt_regs *ctx)
{
	char buf[BUFSIZE]; // PATHLEN is defined to 256
	//bpf_probe_read(buf, sizeof(buf), (void *)ctx->di);
  bpf_probe_read_str(buf, sizeof(buf), (void *)ctx->di);
}
*/

char _license[] SEC("license") = "GPL";
// this number will be interpreted by gobpf-elf-loader to set the current
// running kernel version
__u32 _version SEC("version") = 0xFFFFFFFE;