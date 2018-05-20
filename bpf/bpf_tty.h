#include <linux/types.h>

#define BUFSIZE 256

struct tty_write_t {
    __u32 count;
    char buf[BUFSIZE];
    __u64 timestamp;
    __u64 ino;
    __u64 mnt_ns_inum;
};