// disable randomised task struct (Linux 4.13)
#define randomized_struct_fields_start  struct {
#define randomized_struct_fields_end    };

#include <linux/kconfig.h>
#include <linux/ptrace.h>
#include <linux/version.h>
#include <linux/bpf.h>

#include "bpf_helpers.h"
#include "bpf_tty.h"

// define maps
struct bpf_map_def SEC("maps/excluded_sids") excluded_sids = {
	.type = BPF_MAP_TYPE_HASH,
	.key_size = sizeof(struct sid_t),
	.value_size = sizeof(int),
	.max_entries = 32,
	.pinning = 0,
	.namespace = "",
};

struct bpf_map_def SEC("maps/active_sid") active_sid = {
        .type = BPF_MAP_TYPE_HASH,
        .key_size = sizeof(int),
	.value_size = sizeof(struct sid_t),
        .max_entries = 1,
        .pinning = 0,
        .namespace = "",
};

struct bpf_map_def SEC("maps/available_sids") available_sids = {
	.type = BPF_MAP_TYPE_HASH,
        .key_size = sizeof(struct sid_t),
        .value_size = sizeof(uint64_t),
        .max_entries = 64,
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
    int session_id;

    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&group_leader, sizeof(group_leader), &task->group_leader);
    bpf_probe_read(&pid_link, sizeof(pid_link), group_leader->pids + PIDTYPE_SID);
    bpf_probe_read(&pid, sizeof(pid), pid_link.pid);
    session_id = pid.numbers[0].nr;

    // build sid struct
    struct sid_t sid;
    sid.sid = session_id;

    // add to available sids
    uint64_t time_ns = bpf_ktime_get_ns();
    // update map - there is no point in checking return value for error
    bpf_map_update_elem(&available_sids, &sid, &time_ns, BPF_ANY);

    // if sid is excluded then return
    u64 *exists = bpf_map_lookup_elem(&excluded_sids, &sid);
    if (exists) {
        return 0;
    }

    // return if not active sid
    int key = 0;
    struct sid_t *active = (struct sid_t *)bpf_map_lookup_elem(&active_sid, &key);
    if (!active) {
        // no active sid so return
	return 0;
    } else {
        // if active sid is non-zero and not equal to current sid then return
        int active_session_id = (*active).sid;
        if (active_session_id != 0 && active_session_id != session_id) {
            return 0;
	}
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
    tty_write.sessionid = (unsigned int)session_id;
    tty_write.timestamp = bpf_ktime_get_ns();
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));

    return 0;
}

char _license[] SEC("license") = "GPL";
u32 _version SEC("version") = LINUX_VERSION_CODE;
