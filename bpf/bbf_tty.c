#include <node_config.h>
#include <netdev_config.h>
#include <filter_config.h>

#include <bpf/api.h>

#include <stdint.h>
#include <stdio.h>

#include <linux/bpf.h>
#include <linux/if_ether.h>

#include "lib/utils.h"
#include "lib/common.h"
#include "lib/maps.h"
#include "lib/xdp.h"
#include "lib/eps.h"
#include "lib/events.h"

// define structures
enum pid_type
{
	PIDTYPE_PID,
	PIDTYPE_PGID,
	PIDTYPE_SID,
	PIDTYPE_MAX,
	// only valid to __task_pid_nr_ns() 
	__PIDTYPE_TGID
};
struct upid {
  int nr;
};
struct pid
{
  struct upid numbers[1];
};
struct pid_link
{
  struct pid *pid;
};
struct task_group {
};
struct task_struct {
  struct task_struct *group_leader;
  struct pid_link			pids[PIDTYPE_MAX];
};
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
struct bpf_elf_map __section_maps active_sids = {
	.type		= BPF_MAP_TYPE_HASH,
	.size_key	= sizeof(struct sid_t),
	.size_value	= sizeof(uint64_t),
};

struct bpf_elf_map __section_maps tty_writes = {
	.type		= BPF_MAP_TYPE_PERF_EVENT_ARRAY,
};

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

//int kprobe__tty_write(struct pt_regs *ctx, struct file *file, const char __user *buf, size_t count)
int kprobe__tty_write(struct pt_regs *ctx, struct file *file, const char *buf, size_t count)
{
    struct task_struct *task;
    struct pid_link pid_link;
    struct pid pid;
    int sessionid;
    
    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&pid_link, sizeof(pid_link), (void *)&task->group_leader->pids[PIDTYPE_SID]);
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
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)buf);
    if (count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = count;
    }
    
    // add sessionid to tty_write structure and submit
    tty_write.sessionid = sessionid;
    bpf_perf_event_output(ctx, &tty_write, sizeof(tty_write));
    
    return 0;
}