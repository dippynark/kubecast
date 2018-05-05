// disable randomised task struct (Linux 4.13)
#define randomized_struct_fields_start  struct {
#define randomized_struct_fields_end    };

#include <linux/kconfig.h>
#include <linux/ptrace.h>
#include <linux/version.h>
#include <linux/bpf.h>
#include <linux/fs.h>

#include "bpf_helpers.h"
#include "bpf_tty.h"

// define maps
struct bpf_map_def SEC("maps/excluded_ttys") excluded_ttys = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(struct tty_t),
    .value_size = sizeof(int),
    .max_entries = 32,
    .pinning = 0,
    .namespace = "",
};

struct bpf_map_def SEC("maps/active_tty") active_tty = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(int),
    .value_size = sizeof(struct tty_t),
    .max_entries = 1,
    .pinning = 0,
    .namespace = "",
};

struct bpf_map_def SEC("maps/available_ttys") available_ttys = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(struct tty_t),
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
    unsigned long tty_ino;
    struct inode *f_inode;
    struct file *file;
    struct tty_write_t tty_write;

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
    file = ctx->di;
    bpf_probe_read(&f_inode, sizeof(f_inode), (void *)file->f_inode);
    bpf_probe_read(&tty_ino, sizeof(tty_ino), (void *)f_inode->i_ino);

    // build tty struct
    struct tty_t tty;
    tty.ino = tty_ino;

    // add to available ttys - there is no point in checking return value for error
    uint64_t time_ns = bpf_ktime_get_ns();
    bpf_map_update_elem(&available_ttys, &tty, &time_ns, BPF_ANY);

    // if tty is excluded then return
    u64 *exists = bpf_map_lookup_elem(&excluded_ttys, &tty);
    if (exists) {
        return 0;
    }

    // return if not active tty
    int key = 0;
    struct tty_t *active_tty = (struct tty_t *)bpf_map_lookup_elem(&active_tty, &key);
    if (!active_tty) {
        // no active tty so return
	      return 0;
    } else {
        // if active tty is non-zero and not equal to current sid then return
        unsigned long active_tty_ino = (*active_tty).ino;
        if (active_tty_ino != tty_ino) {
            return 0;
	      }
    }

    // bpf_probe_read() can only use a fixed size, so truncate to count in user space:
    struct tty_write_t tty_write;
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)ctx->si);

    int tty_write_count = ctx->dx;
    if (tty_write_count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = tty_write_count;
    }

    // submit tty_write event
    tty_write.timestamp = bpf_ktime_get_ns();
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));

    return 0;
}

char _license[] SEC("license") = "GPL";
u32 _version SEC("version") = LINUX_VERSION_CODE;
