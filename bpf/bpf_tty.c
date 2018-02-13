// disable randomised task struct
#define randomized_struct_fields_start  struct {
#define randomized_struct_fields_end    };

#include <linux/kconfig.h>

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wgnu-variable-sized-type-not-at-end"
#pragma clang diagnostic ignored "-Waddress-of-packed-member"
#include <linux/ptrace.h>
#pragma clang diagnostic pop
#include <linux/version.h>
#include <linux/bpf.h>
#include "bpf_helpers.h"
#include "bpf_tty.h"

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wtautological-compare"
#pragma clang diagnostic ignored "-Wgnu-variable-sized-type-not-at-end"
#pragma clang diagnostic ignored "-Wenum-conversion"
#include <net/sock.h>
#pragma clang diagnostic pop
#include <net/inet_sock.h>
#include <net/net_namespace.h>

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
	.value_size = sizeof(__u32),
	.max_entries = 1024,
	.pinning = 0,
	.namespace = "",
};


// save_sid saves a sessionid generated from a call
// to setsid to the active_sids map
int save_sid(struct pt_regs *ctx) {

    struct sid_t sid_struct = {};
    int sid = PT_REGS_RC(ctx);
    uint64_t time_ns = bpf_ktime_get_ns();

    sid_struct.sid = sid;

    // BPF_ANY: create new element or update existing
    bpf_map_update_elem(&active_sids, &sid_struct, &time_ns, BPF_ANY);

    return 0;

}

SEC("kprobe/tty_write")
int kprobe__tty_write(struct pt_regs *ctx)
{
    struct task_struct *task;
    struct task_struct *group_leader;
    struct pid_link pid_link;
    struct upid upid;
    int sessionid;

    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&group_leader, sizeof(group_leader), (void *)task->group_leader);
    bpf_probe_read(&pid_link, sizeof(pid_link), (void *)(group_leader->pids + PIDTYPE_SID));
    bpf_probe_read(&upid, sizeof(upid), (void *)pid_link.pid->numbers);
    sessionid = upid.nr;

    /*if(sessionid == current_pid) {
      // this is the session leader so return
      return 0;
    }*/

    // build session struct key
    struct sid_t sid_key;
    sid_key.sid = sessionid;

    // if sid does not exist in our map then return
    /*u64 *time_ns = bpf_map_lookup_elem(&active_sids, &sid_key);
    if (!time_ns) {
        return 0;
    }*/

    if(sessionid == 0) {
      return 0;
    }

    // bpf_probe_read() can only use a fixed size, so truncate to count
    // in user space:
    // we use the following mapping of registers to arguments
    /*
    R0 – rax      return value from function
    R1 – rdi      1st argument
    R2 – rsi      2nd argument
    R3 – rdx      3rd argument
    R4 – rcx      4th argument
    R5 – r8       5th argument
    R6 – rbx      callee saved
    R7 - r13      callee saved
    R8 - r14      callee saved
    R9 - r15      callee saved
    R10 – rbp     frame pointer
    */
    struct tty_write_t tty_write;
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)ctx->si);

    int tty_write_count = ctx->dx;
    if (tty_write_count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = tty_write_count;
    }

    // add sessionid to tty_write structure and submit
    tty_write.sessionid = sessionid;
    tty_write.timestamp = bpf_ktime_get_ns();
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));

    return 0;
}

char _license[] SEC("license") = "GPL";
// this number will be interpreted by gobpf-elf-loader to set the current
// running kernel version
__u32 _version SEC("version") = 0xFFFFFFFE;
