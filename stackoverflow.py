#!/usr/bin/python

from bcc import BPF
import ctypes as ct
import os
import threading
import time
import sys

prog=r"""
#include <uapi/linux/ptrace.h>

#include <linux/sched.h>
#include <linux/fs.h>
#include <linux/nsproxy.h>
#include <linux/ns_common.h>

#define BUFSIZE 256
struct tty_write_t {
    int count;
    char buf[BUFSIZE];
    unsigned int sessionid;
};

// define maps
BPF_PERF_OUTPUT(tty_writes);

int kprobe__tty_write(struct pt_regs *ctx, struct file *file,
    const char __user *buf, size_t count)
{
    struct task_struct *task;
    struct task_struct *group_leader;
    struct pid_link pid_link;
    struct pid pid;
    int sessionid;

    // get current sessionid
    task = (struct task_struct *)bpf_get_current_task();
    bpf_probe_read(&group_leader, sizeof(group_leader), (void *)&task->group_leader);
    bpf_probe_read(&pid_link, sizeof(pid_link), (void *)group_leader->pids + PIDTYPE_SID);
    bpf_probe_read(&pid, sizeof(pid), pid_link.pid);
    sessionid = pid.numbers[0].nr;

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

"""

b = BPF(text=prog)

BUFSIZE = 256
class TTYWrite(ct.Structure):
    _fields_ = [
        ("count", ct.c_int),
        ("buf", ct.c_char * BUFSIZE),
        ("sessionid", ct.c_int)
    ]

# process tty_write
def print_tty_write(cpu, data, size):
    tty_write = ct.cast(data, ct.POINTER(TTYWrite)).contents
    print(str(tty_write.sessionid))

b["tty_writes"].open_perf_buffer(print_tty_write)
while 1:
    b.kprobe_poll()
