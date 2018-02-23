// disable randomised task struct (Linux 4.13)
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
struct bpf_map_def SEC("maps/excluded_sids") excluded_sids = {
	.type = BPF_MAP_TYPE_HASH,
	.key_size = sizeof(struct sid_t),
	.value_size = sizeof(uint64_t),
	.max_entries = 10,
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

SEC("kprobe/tty_write")
int kprobe__tty_write(struct pt_regs *ctx)
{
    struct task_struct *task;
    struct task_struct *group_leader;
    struct pid_link pid_link;
    struct pid pid;
    int sessionid;

    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&group_leader, sizeof(group_leader), &task->group_leader);
    bpf_probe_read(&pid_link, sizeof(pid_link), group_leader->pids + PIDTYPE_SID);
    bpf_probe_read(&pid, sizeof(pid), pid_link.pid);
    sessionid = pid.numbers[0].nr;

    // build session struct key
    struct sid_t sid_key;
    sid_key.sid = sessionid;

    // if sid does not exist in our map then return
    u64 *exists = bpf_map_lookup_elem(&excluded_sids, &sid_key);
    if (exists) {
        return 0;
    }

    // bpf_probe_read() can only use a fixed size, so truncate to count
    // in user space:
    struct tty_write_t tty_write;
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)ctx->si);

    int tty_write_count = ctx->dx;
    if (tty_write_count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = tty_write_count;
    }

    // add sessionid to tty_write structure and submit
    tty_write.sessionid = (unsigned int)sessionid;
    tty_write.timestamp = bpf_ktime_get_ns();
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));

    return 0;
}

char _license[] SEC("license") = "GPL";
// this number will be interpreted by gobpf-elf-loader to set the current
// running kernel version
__u32 _version SEC("version") = 0xFFFFFFFE;
