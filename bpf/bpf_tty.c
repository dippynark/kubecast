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
    struct tty_write_t tty_write = {};

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
    file = (struct file *)ctx->di;
    bpf_probe_read(&f_inode, sizeof(f_inode), (void *)&file->f_inode);
    bpf_probe_read(&tty_ino, sizeof(tty_ino), (void *)&f_inode->i_ino);

    // bpf_probe_read() can only use a fixed size, so truncate to count in user space:
    bpf_probe_read(&tty_write.buf, BUFSIZE, (void *)ctx->si);

    int tty_write_count = ctx->dx;
    if (tty_write_count > BUFSIZE) {
        tty_write.count = BUFSIZE;
    } else {
        tty_write.count = tty_write_count;
    }

    // submit tty_write event
    tty_write.ino = (unsigned long) tty_ino;
    tty_write.timestamp = bpf_ktime_get_ns();
    bpf_perf_event_output(ctx, &tty_writes, 0, &tty_write, sizeof(tty_write));

    return 0;
}

char _license[] SEC("license") = "GPL";
u32 _version SEC("version") = LINUX_VERSION_CODE;
